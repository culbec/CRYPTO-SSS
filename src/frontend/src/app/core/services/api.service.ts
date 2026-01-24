import { Injectable } from '@angular/core';
import { HttpClient, HttpParams } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  LoginRequest,
  RegisterRequest,
  AuthResponse
} from '../models/auth.model';
import {
  Poll,
  PollStatus,
  BallotResponse,
  ShareDistribution,
  ShareStatus,
  CreatePollRequest
} from '../models/poll.model';

@Injectable({
  providedIn: 'root'
})
export class ApiService {
  private readonly baseUrl = 'http://localhost:3000/api';

  constructor(private http: HttpClient) {}

  // Auth endpoints
  register(request: RegisterRequest): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.baseUrl}/auth/register`, request);
  }

  login(request: LoginRequest): Observable<AuthResponse> {
    return this.http.post<AuthResponse>(`${this.baseUrl}/auth/login`, request);
  }

  logout(): Observable<{ message: string }> {
    return this.http.post<{ message: string }>(`${this.baseUrl}/auth/logout`, {});
  }

  validateToken(): Observable<{ message: string }> {
    return this.http.post<{ message: string }>(`${this.baseUrl}/auth/validate`, {});
  }

  // Poll endpoints
  listPolls(status?: PollStatus): Observable<{ polls: Poll[]; total: number }> {
    let params = new HttpParams();
    if (status) {
      params = params.set('status', status);
    }
    return this.http.get<{ polls: Poll[]; total: number }>(`${this.baseUrl}/polls`, { params });
  }

  getPoll(id: string): Observable<Poll> {
    return this.http.get<Poll>(`${this.baseUrl}/polls/${id}`);
  }

  createPoll(request: CreatePollRequest): Observable<Poll> {
    return this.http.post<Poll>(`${this.baseUrl}/polls`, request);
  }

  updatePollStatus(id: string, status: PollStatus): Observable<Poll> {
    return this.http.put<Poll>(`${this.baseUrl}/polls/${id}/status`, { status });
  }

  freezePoll(id: string): Observable<{ message: string }> {
    return this.http.post<{ message: string }>(`${this.baseUrl}/polls/${id}/freeze`, {});
  }

  getMyShare(pollId: string): Observable<ShareDistribution> {
    return this.http.get<ShareDistribution>(`${this.baseUrl}/polls/${pollId}/my-share`);
  }

  getShareStatus(pollId: string): Observable<ShareStatus> {
    return this.http.get<ShareStatus>(`${this.baseUrl}/polls/${pollId}/share-status`);
  }

  // Ballot endpoints
  castBallot(pollId: string, encryptedVote: string): Observable<BallotResponse> {
    return this.http.post<BallotResponse>(`${this.baseUrl}/ballots`, {
      poll_id: pollId,
      encrypted_vote: encryptedVote
    });
  }

  getMyBallot(pollId: string): Observable<BallotResponse> {
    return this.http.get<BallotResponse>(`${this.baseUrl}/ballots/poll/${pollId}/my-ballot`);
  }

  contributeShare(pollId: string, shareValue: string): Observable<{ message: string }> {
    return this.http.post<{ message: string }>(`${this.baseUrl}/ballots/contribute-share`, {
      poll_id: pollId,
      share_value: shareValue
    });
  }

  // Admin endpoints
  testSSS(secret: string, threshold: number, total: number): Observable<any> {
    return this.http.post(`${this.baseUrl}/admin/sss-healthcheck`, {
      secret,
      threshold,
      total
    });
  }

  testAccessStructure(): Observable<any> {
    return this.http.get(`${this.baseUrl}/admin/sss-access-test`);
  }

  listUsers(role?: string): Observable<any[]> {
    let params = new HttpParams();
    if (role) {
      params = params.set('role', role);
    }
    return this.http.get<any[]>(`${this.baseUrl}/admin/users`, { params });
  }

  updateUserRole(userId: string, role: string): Observable<any> {
    return this.http.put(`${this.baseUrl}/admin/users/${userId}/role`, {}, {
      params: new HttpParams().set('role', role)
    });
  }

  // Health check
  healthCheck(): Observable<{ message: string }> {
    return this.http.get<{ message: string }>(`${this.baseUrl}/health`);
  }

  ping(): Observable<{ message: string }> {
    return this.http.get<{ message: string }>(`${this.baseUrl}/ping`);
  }
}
