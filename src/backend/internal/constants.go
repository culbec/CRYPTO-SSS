package internal

import "time"

// ////////////////////////////
// LOGGING CONSTANTS
// ////////////////////////////
const LOG_FILE string = "logs/backend.log"

// ////////////////////////////
// SERVER CONSTANTS
// ////////////////////////////
const SERVER_HOST string = "127.0.0.1"
const SERVER_PORT string = "3000"

// ////////////////////////////
// SECURITY CONSTANTS
// ////////////////////////////
const ARGON2ID_DEFAULT_TIME uint32 = 5
const ARGON2ID_DEFAULT_MEMORY uint32 = 7 * 1024
const ARGON2ID_DEFAULT_THREADS uint8 = 4
const ARGON2ID_DEFAULT_KEY_LEN uint32 = 32
const ARGON2ID_DEFAULT_SALT_LEN uint32 = 16

const DEFAULT_JWT_EXPIRY time.Duration = 60 * time.Minute

// ////////////////////////////
// CONFIG CONSTANTS
// ////////////////////////////
const CONFIG_FILE string = "configs/config.json"
const CONFIG_FILE_LOCAL string = "configs/config.local.json"

//////////////////////////////
// FORMATS
// ////////////////////////////

const TIME_FORMAT string = "2006-01-02T15:04:05.000Z"
