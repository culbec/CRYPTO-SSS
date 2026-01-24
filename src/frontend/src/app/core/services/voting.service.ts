import { Injectable, inject } from '@angular/core';
import { HttpClient } from '@angular/common/http';
import { Observable } from 'rxjs';
import {
  Poll,
  BallotResponse,
  ShareDistribution,
  ShareStatus,
  PollResultResponse
} from '../models/poll.model';

@Injectable({
  providedIn: 'root'
})
export class VotingService {
  private http = inject(HttpClient);
  private readonly baseUrl = 'http://localhost:3000/api';

  // Get all polls
  getPolls(): Observable<{ polls: Poll[] }> {
    return this.http.get<{ polls: Poll[] }>(`${this.baseUrl}/polls`);
  }

  // Get single poll by ID
  getPollById(pollId: string): Observable<Poll> {
    return this.http.get<Poll>(`${this.baseUrl}/polls/${pollId}`);
  }

  // Create a new poll
  createPoll(pollData: any): Observable<Poll> {
    return this.http.post<Poll>(`${this.baseUrl}/polls`, pollData);
  }

  // Update poll status
  updatePollStatus(pollId: string, status: string): Observable<Poll> {
    return this.http.put<Poll>(`${this.baseUrl}/polls/${pollId}/status`, {
      status: status
    });
  }

  // Freeze poll and distribute shares
  freezePoll(pollId: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/polls/${pollId}/freeze`, {});
  }

  // Cast an encrypted ballot
  castBallot(pollId: string, encryptedVote: string): Observable<BallotResponse> {
    return this.http.post<BallotResponse>(`${this.baseUrl}/ballots`, {
      poll_id: pollId,
      encrypted_vote: encryptedVote
    });
  }

  // Get user's ballot receipt for a poll
  getMyBallot(pollId: string): Observable<BallotResponse> {
    return this.http.get<BallotResponse>(`${this.baseUrl}/ballots/poll/${pollId}/my-ballot`);
  }

  // Contribute share for reveal phase
  contributeShare(pollId: string, shareValue: string): Observable<any> {
    return this.http.post(`${this.baseUrl}/ballots/contribute-share`, {
      poll_id: pollId,
      share_value: shareValue
    });
  }

  // Get share status (how many shares collected)
  getShareStatus(pollId: string): Observable<ShareStatus> {
    return this.http.get<ShareStatus>(`${this.baseUrl}/polls/${pollId}/share-status`);
  }

  // Get user's share for a poll
  getMyShare(pollId: string): Observable<ShareDistribution> {
    return this.http.get<ShareDistribution>(`${this.baseUrl}/polls/${pollId}/my-share`);
  }

  // Reveal results (reconstruct master key and decrypt votes)
  revealResults(pollId: string): Observable<PollResultResponse> {
    return this.http.post<PollResultResponse>(`${this.baseUrl}/polls/${pollId}/reveal`, {});
  }

  // Get poll results
  getPollResults(pollId: string): Observable<PollResultResponse> {
    return this.http.get<PollResultResponse>(`${this.baseUrl}/polls/${pollId}/results`);
  }
}

