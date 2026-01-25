import { Injectable, signal, computed, effect } from '@angular/core';
import { Router } from '@angular/router';
import { ApiService } from './api.service';
import {
  LoginRequest,
  RegisterRequest,
  AuthResponse,
  User,
  UserRole
} from '../models/auth.model';

interface AuthState {
  user: User | null;
  token: string | null;
  isAuthenticated: boolean;
  isLoading: boolean;
  error: string | null;
}

@Injectable({
  providedIn: 'root'
})
export class AuthService {
  private readonly state = signal<AuthState>({
    user: null,
    token: null,
    isAuthenticated: false,
    isLoading: false,
    error: null
  });

  // Computed signals
  readonly user = computed(() => this.state().user);
  readonly token = computed(() => this.state().token);
  readonly isAuthenticated = computed(() => this.state().isAuthenticated);
  readonly isLoading = computed(() => this.state().isLoading);
  readonly error = computed(() => this.state().error);
  readonly userRole = computed(() => this.state().user?.role);

  constructor(
    private apiService: ApiService,
    private router: Router
  ) {
    this.initializeAuth();
    this.setupTokenPersistence();
  }

  /**
   * Initialize authentication from localStorage
   */
  private initializeAuth(): void {
    const token = localStorage.getItem('auth_token');
    const user = localStorage.getItem('auth_user');

    if (token && user) {
      try {
        const parsedUser = JSON.parse(user);
        this.state.update(s => ({
          ...s,
          token,
          user: parsedUser,
          isAuthenticated: true
        }));
      } catch (e) {
        this.clearAuthState();
      }
    }
  }

  /**
   * Clear authentication state
   */
  private clearAuthState(): void {
    this.state.set({
      user: null,
      token: null,
      isAuthenticated: false,
      isLoading: false,
      error: null
    });
  }

  /**
   * Setup effect to persist token changes to localStorage
   */
  private setupTokenPersistence(): void {
    effect(() => {
      const token = this.token();
      const user = this.user();

      if (token && user) {
        localStorage.setItem('auth_token', token);
        localStorage.setItem('auth_user', JSON.stringify(user));
      } else {
        localStorage.removeItem('auth_token');
        localStorage.removeItem('auth_user');
      }
    });
  }

  /**
   * Register a new user
   */
  register(request: RegisterRequest): Promise<void> {
    this.state.update(s => ({ ...s, isLoading: true, error: null }));

    return new Promise((resolve, reject) => {
      this.apiService.register(request).subscribe({
        next: (response: AuthResponse) => {
          this.handleAuthResponse(response);
          this.router.navigate(['/dashboard']);
          resolve();
        },
        error: (error) => {
          const errorMessage = error?.error?.error || 'Registration failed';
          this.state.update(s => ({ ...s, isLoading: false, error: errorMessage }));
          reject(error);
        }
      });
    });
  }

  /**
   * Login user
   */
  login(request: LoginRequest): Promise<void> {
    this.state.update(s => ({ ...s, isLoading: true, error: null }));

    return new Promise((resolve, reject) => {
      this.apiService.login(request).subscribe({
        next: (response: AuthResponse) => {
          this.handleAuthResponse(response);
          this.router.navigate(['/dashboard']);
          resolve();
        },
        error: (error) => {
          const errorMessage = error?.error?.error || 'Login failed';
          this.state.update(s => ({ ...s, isLoading: false, error: errorMessage }));
          reject(error);
        }
      });
    });
  }

  /**
   * Logout user
   */
  logout(): void {
    this.state.update(s => ({ ...s, isLoading: true }));

    this.apiService.logout().subscribe({
      next: () => this.completeLogout(),
      error: () => this.completeLogout()
    });
  }

  /**
   * Handle logout completion
   */
  private completeLogout(): void {
    this.clearAuthState();
    this.router.navigate(['/home']);
  }

  /**
   * Clear error message
   */
  clearError(): void {
    this.state.update(s => ({ ...s, error: null }));
  }

  /**
   * Check if user has a specific role
   */
  hasRole(role: UserRole | UserRole[]): boolean {
    const userRole = this.userRole();
    if (!userRole) return false;

    if (Array.isArray(role)) {
      return role.includes(userRole);
    }
    return userRole === role;
  }

  /**
   * Handle authentication response
   */
  private handleAuthResponse(response: AuthResponse): void {
    const user: User = {
      id: response.user_id,
      username: response.username,
      role: response.role,
      date: new Date().toISOString()
    };

    this.state.update(s => ({
      ...s,
      token: response.token,
      user,
      isAuthenticated: true,
      isLoading: false,
      error: null
    }));
  }
}
