import { Routes } from '@angular/router';
import { authRoutes } from './features/auth/auth.routes';
import { homeRoutes } from './features/home/home.routes';
import { dashboardRoutes } from './features/dashboard/dashboard.routes';
import { authGuardFactory } from './core/guards/auth.guard';
import { AuthService } from './core/services/auth.service';
import { Router } from '@angular/router';
import { inject } from '@angular/core';
import { roleGuardFactory } from './core/guards/role.guard';

// Create a guard using the factory pattern
const authGuard = () => {
  const authService = inject(AuthService);
  const router = inject(Router);
  return authGuardFactory(authService, router);
};

const roleGuard = () => {
  const authService = inject(AuthService);
  const router = inject(Router);
  return roleGuardFactory(authService, router);
}

export const routes: Routes = [
  {
    path: '',
    redirectTo: 'home',
    pathMatch: 'full',
  },
  {
    path: 'home',
    children: homeRoutes
  },
  {
    path: 'auth',
    children: authRoutes
  },
  {
    path: 'dashboard',
    canActivate: [authGuard, roleGuard],
    data: { roles: ['admin', 'official', 'auditor', 'voter'] },
    children: dashboardRoutes,
  },
  {
    path: '**',
    redirectTo: 'home',
  }
];
