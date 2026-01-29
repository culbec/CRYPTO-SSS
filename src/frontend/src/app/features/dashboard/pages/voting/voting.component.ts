import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { VotingService } from '../../../../core/services/voting.service';
import { AuthService } from '../../../../core/services/auth.service';
import { PollEventsService } from '../../../../core/services/poll-events.service';
import { Poll, PollOption, BallotResponse } from '../../../../core/models/poll.model';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

@Component({
  selector: 'app-voting',
  standalone: true,
  imports: [CommonModule, RouterLink],
  templateUrl: './voting.component.html',
  styleUrls: ['./voting.component.scss']
})
export class VotingComponent implements OnInit, OnDestroy {
  private route = inject(ActivatedRoute);
  private votingService = inject(VotingService);
  private authService = inject(AuthService);
  private router = inject(Router);
  private eventsService = inject(PollEventsService);
  private destroy$ = new Subject<void>();

  poll: Poll | null = null;
  selectedOption: PollOption | null = null;
  myBallot: BallotResponse | null = null;
  pollId = '';

  showModal = false;
  modalType: 'success' | 'error' = 'success';
  modalTitle = '';
  modalMessage = '';

  ngOnInit() {
    this.pollId = this.route.snapshot.paramMap.get('id') || '';
    if (this.pollId) {
      this.loadPoll(this.pollId);
      this.checkExistingBallot(this.pollId);
      this.setupRealtimeListeners();
    }
  }

  ngOnDestroy() {
    this.destroy$.next();
    this.destroy$.complete();
  }

  private setupRealtimeListeners(): void {
    // If poll status changes while viewing, reload the poll
    this.eventsService.onPollStatusChanged(this.pollId)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.loadPoll(this.pollId);
      });
  }

  loadPoll(pollId: string) {
    this.votingService.getPollById(pollId).subscribe({
      next: (poll) => this.poll = poll,
      error: (err: any) => {
        console.error('Failed to load poll', err);
        this.router.navigate(['/dashboard/polls']);
      }
    });
  }

  checkExistingBallot(pollId: string) {
    this.votingService.getMyBallot(pollId).subscribe({
      next: (ballot) => this.myBallot = ballot,
      error: (err: any) => {
        if (err.status !== 404) {
          console.error('Error checking ballot', err);
        }
      }
    });
  }

  selectOption(option: PollOption) {
    this.selectedOption = option;
  }

  castVote() {
    if (!this.selectedOption || !this.poll) return;

    // TODO: use proper encryption (AES-256-GCM or NaCl)
    const encryptedVote = btoa(JSON.stringify({
      option_id: this.selectedOption.id,
      voter_id: this.authService.user()?.id,
      timestamp: new Date().toISOString()
    }));

    this.votingService.castBallot(this.poll.id, encryptedVote).subscribe({
      next: (response) => {
        this.myBallot = response;
        this.modalType = 'success';
        this.modalTitle = 'Vote cast successfully';
        this.modalMessage = '✓ Your vote was submitted and a receipt was generated.';
        this.showModal = true;
      },
      error: (err: any) => {
        console.error('Failed to cast vote', err);
        this.modalType = 'error';
        this.modalTitle = 'Could not cast vote';
        this.modalMessage = err.error?.error || 'Unknown error';
        this.showModal = true;
      }
    });
  }

  getStatusLabel(status: string | undefined): string {
    return {
      draft: 'Draft',
      open: 'Open',
      closed: 'Closed',
      frozen: 'Frozen',
      revealed: 'Revealed'
    }[status || ''] || status || '';
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  closeModal() {
    this.showModal = false;
  }
}
