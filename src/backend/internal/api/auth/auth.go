package auth

import (
	"encoding/hex"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	constants "github.com/culbec/CRYPTO-sss/src/backend/internal"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/logging"
	"github.com/culbec/CRYPTO-sss/src/backend/internal/types"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/mongo"
	"github.com/culbec/CRYPTO-sss/src/backend/pkg/security"
	security_jwt "github.com/culbec/CRYPTO-sss/src/backend/pkg/security/jwt"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson"
)

const authHeaderPrefix string = "Bearer "

type tokenManager struct {
	tokenBlacklist map[string]time.Time
	blacklistMutex *sync.RWMutex
}

type AuthHandler struct {
	db           *mongo.Client
	hasher       *security.Argon2idHash
	jwtManager   *security_jwt.JWTManager
	tokenManager *tokenManager
}

func newTokenManager() *tokenManager {
	return &tokenManager{
		tokenBlacklist: make(map[string]time.Time),
		blacklistMutex: &sync.RWMutex{},
	}
}

func (t *tokenManager) addToBlacklistUntil(token string, expiresAt time.Time) {
	t.blacklistMutex.Lock()
	defer t.blacklistMutex.Unlock()
	t.tokenBlacklist[token] = expiresAt
}

func (t *tokenManager) isBlacklisted(token string) bool {
	now := time.Now()

	t.blacklistMutex.RLock()
	exp, ok := t.tokenBlacklist[token]
	if !ok {
		t.blacklistMutex.RUnlock()
		return false
	}

	// blacklisted while token is not yet expired
	if now.Before(exp) {
		t.blacklistMutex.RUnlock()
		return true
	}
	t.blacklistMutex.RUnlock()

	// opportunistic cleanup
	t.blacklistMutex.Lock()
	if exp2, ok2 := t.tokenBlacklist[token]; ok2  && !now.Before(exp2) {
		delete(t.tokenBlacklist, token)
	}
	t.blacklistMutex.Unlock()
	return false
}

func NewAuthHandler(db *mongo.Client, secretKey []byte) *AuthHandler {
	hasher := security.NewArgon2idHash(
		constants.ARGON2ID_DEFAULT_TIME,
		constants.ARGON2ID_DEFAULT_MEMORY,
		constants.ARGON2ID_DEFAULT_THREADS,
		constants.ARGON2ID_DEFAULT_KEY_LEN,
		constants.ARGON2ID_DEFAULT_SALT_LEN,
	)

	jwtManager := security_jwt.NewJWTManager(secretKey, constants.DEFAULT_JWT_EXPIRY)
	tokenManager := newTokenManager()

	return &AuthHandler{
		db:           db,
		hasher:       hasher,
		jwtManager:   jwtManager,
		tokenManager: tokenManager,
	}
}

func (a *AuthHandler) GetJwtManager() *security_jwt.JWTManager {
	return a.jwtManager
}

func (a *AuthHandler) GetTokenManager() *tokenManager {
	return a.tokenManager
}

func (a *AuthHandler) ValidateToken(ctx *gin.Context) (string, error) {
	logger := logging.FromContext(ctx.Request.Context())

	token := ctx.GetHeader("Authorization")
	if token == "" {
		msg := "no authorization token provided"
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return "", errors.New(msg)
	}

	if !strings.HasPrefix(token, authHeaderPrefix) {
		msg := "invalid authorization token format. Expected " + authHeaderPrefix + "<token>"
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return "", errors.New(msg)
	}

	token = strings.TrimPrefix(token, authHeaderPrefix)

	if a.tokenManager.isBlacklisted(token) {
		msg := "authorization token is blacklisted"
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": msg})
		return "", errors.New(msg)
	}

	username, _, err := a.jwtManager.ValidateToken(token)
	if err != nil {
		msg := "invalid authorization token: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": msg})
		return "", errors.New(msg)
	}

	logger.Info("authorization token validated successfully for user: " + username)
	return username, nil
}

func (a *AuthHandler) Login(ctx *gin.Context) error {
	logger := logging.FromContext(ctx.Request.Context())

	var req types.LoginRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid login request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return errors.New(msg)
	}

	var user []types.User
	if status, err := a.db.QueryCollection(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.UserCollection],
		&bson.D{{Key: "username", Value: req.Username}},
		nil,
		&user,
	); err != nil {
		msg := "error querying user: " + err.Error()
		logger.Error(msg)
		ctx.JSON(status, gin.H{"error": msg})
		return errors.New(msg)
	}

	if len(user) == 0 {
		msg := "user with username '" + req.Username + "' not found"
		logger.Error(msg)
		ctx.JSON(http.StatusNotFound, gin.H{"error": msg})
		return errors.New(msg)
	}

	saltBytes, decodeErr := hex.DecodeString(user[0].Salt)
	if decodeErr != nil {
		// Backward compatibility with previously stored raw-string salts.
		saltBytes = []byte(user[0].Salt)
	}

	err := a.hasher.ComparePasswords(
		[]byte(req.Password),
		saltBytes,
		[]byte(user[0].Password),
	)
	if err != nil {
		msg := "invalid password for user '" + req.Username + "'"
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": msg})
		return errors.New(msg)
	}

	token, err := a.jwtManager.GenerateToken(user[0].Username)
	if err != nil {
		msg := "error generating token for user '" + req.Username + "'"
		logger.Error(msg)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return errors.New(msg)
	}

	ctx.JSON(http.StatusOK, types.AuthResponse{
		UserID: user[0].ID.Hex(),
		Token:  token,
	})
	return nil
}

func (a *AuthHandler) Logout(ctx *gin.Context) error {
	logger := logging.FromContext(ctx.Request.Context())

	authHeader := ctx.GetHeader("Authorization")

	if authHeader == "" {
		msg := "no authorization token provided"
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return errors.New(msg)
	}

	if !strings.HasPrefix(authHeader, authHeaderPrefix) {
		msg := "invalid authorization token format. Expected " + authHeaderPrefix + "<token>"
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return errors.New(msg)
	}

	token := strings.TrimPrefix(authHeader, authHeaderPrefix)

	if a.tokenManager.isBlacklisted(token) {
		msg := "authorization token is blacklisted"
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": msg})
		return errors.New(msg)
	}

	_, expiresAt, err := a.jwtManager.ValidateToken(token)
	if err != nil {
		msg := "invalid authorization token: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": msg})
		return errors.New(msg)
	}

	a.tokenManager.addToBlacklistUntil(token, expiresAt)

	msg := "logged out successfully"
	logger.Info(msg)
	ctx.JSON(http.StatusOK, gin.H{"message": msg})
	return nil
}

func (a *AuthHandler) Register(ctx *gin.Context) error {
	logger := logging.FromContext(ctx.Request.Context())

	var req types.RegisterRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		msg := "invalid register request: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusBadRequest, gin.H{"error": msg})
		return errors.New(msg)
	}

	hashSalt, err := a.hasher.GenerateHash(
		[]byte(req.Password),
		[]byte{},
	)
	if err != nil {
		msg := "error generating hash for password: " + err.Error()
		logger.Error(msg)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return errors.New(msg)
	}

	objDate := time.Now().Format(constants.TIME_FORMAT)
	objVersion := 1

	user := types.User{
		Username: req.Username,
		Password: string(hashSalt.Hash),
		Salt:     hex.EncodeToString(hashSalt.Salt),
		Date:     objDate,
		Version:  objVersion,
	}

	insertingConditions := bson.D{
		{Key: "username", Value: req.Username},
	}

	id, status, err := a.db.InsertDocument(
		ctx.Request.Context(),
		mongo.DbCollections[mongo.UserCollection],
		&insertingConditions,
		&user,
	)

	if err != nil {
		msg := "error inserting user: " + err.Error()
		logger.Error(msg)
		ctx.JSON(status, gin.H{"error": msg})
		return errors.New(msg)
	}

	if id == nil {
		msg := "user already exists"
		logger.Error(msg)
		ctx.JSON(status, gin.H{"error": msg})
		return errors.New(msg)
	}

	token, err := a.jwtManager.GenerateToken(req.Username)
	if err != nil {
		msg := "error generating token for user '" + req.Username + "'"
		logger.Error(msg)
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": msg})
		return errors.New(msg)
	}

	userId := id.Hex()
	ctx.JSON(http.StatusCreated, types.AuthResponse{
		UserID: userId,
		Token:  token,
	})
	return nil

}
