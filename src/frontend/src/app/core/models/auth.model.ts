export interface LoginRequest {
  username: string;
  password: string;
}

export interface RegisterRequest {
  username: string;
  password: string;
  role: UserRole;
}

export interface AuthResponse {
  user_id: string;
  token: string;
  role: UserRole;
}

export interface User {
  id: string;
  username: string;
  role: UserRole;
  date: string;
}

export type UserRole = 'voter' | 'auditor' | 'official' | 'admin';

export const ROLE_LABELS: Record<UserRole, string> = {
  voter: 'Voter',
  auditor: 'Auditor',
  official: 'Official',
  admin: 'Administrator'
};

export const ROLE_DESCRIPTIONS: Record<UserRole, string> = {
  voter: 'Can cast votes in polls',
  auditor: 'Can audit polls and participate in result reveal',
  official: 'Can create and manage polls',
  admin: 'Full system access and testing capabilities'
};
