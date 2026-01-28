import { Component, OnInit, OnDestroy, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router, RouterLink } from '@angular/router';
import { MatSnackBarModule } from '@angular/material/snack-bar';
import { MatIconModule } from '@angular/material/icon';
import { VotingService } from '../../../../core/services/voting.service';
import { AuthService } from '../../../../core/services/auth.service';
import { NotificationService } from '../../../../core/services/notification.service';
import { PollEventsService } from '../../../../core/services/poll-events.service';
import { Poll, PollStatus } from '../../../../core/models/poll.model';
import { Subject } from 'rxjs';
import { takeUntil } from 'rxjs/operators';

@Component({
  selector: 'app-poll-detail',
  standalone: true,
  imports: [CommonModule, RouterLink, MatSnackBarModule, MatIconModule],
  templateUrl: './poll-detail.component.html',
  styleUrls: ['./poll-detail.component.scss']
})
export class PollDetailComponent implements OnInit, OnDestroy {
  private readonly route = inject(ActivatedRoute);
  private readonly router = inject(Router);
  private readonly votingService = inject(VotingService);
  private readonly authService = inject(AuthService);
  private readonly notificationService = inject(NotificationService);
  private readonly eventsService = inject(PollEventsService);
  private readonly destroy$ = new Subject<void>();

  poll: Poll | null = null;
  loading = true;
  error: string | null = null;
  updatingStatus = false;
  role: 'voter' | 'auditor' | 'official' | 'admin' = 'voter';
  pollId = '';

  ngOnInit(): void {
    this.role = this.authService.userRole() as any;
    this.pollId = this.route.snapshot.paramMap.get('id') || '';
    this.loadPoll();
    this.setupRealtimeListeners();
  }

  ngOnDestroy(): void {
    this.destroy$.next();
    this.destroy$.complete();
  }

  private setupRealtimeListeners(): void {
    // When poll status changes, reload the poll
    this.eventsService.onPollStatusChanged(this.pollId)
      .pipe(takeUntil(this.destroy$))
      .subscribe((newStatus) => {
        this.loadPoll();
      });

    // When results are revealed, reload the poll
    this.eventsService.onResultsRevealed(this.pollId)
      .pipe(takeUntil(this.destroy$))
      .subscribe(() => {
        this.loadPoll();
      });
  }

  private loadPoll(): void {
    const pollId = this.route.snapshot.paramMap.get('id');
    if (!pollId) {
      this.error = 'Poll ID not found';
      this.loading = false;
      return;
    }

    this.votingService.getPollById(pollId).subscribe({
      next: (poll) => {
        this.poll = poll;
        this.loading = false;
      },
      error: (err) => {
        this.error = 'Failed to load poll: ' + (err?.error?.error || err?.message || 'Unknown error');
        this.loading = false;
      }
    });
  }

  get isManager(): boolean {
    return this.role === 'official' || this.role === 'admin';
  }

  get canContribute(): boolean {
    return this.role === 'auditor' || this.role === 'official' || this.role === 'admin';
  }

  get canOpenPoll(): boolean {
    return this.isManager;
  }

  openPollForVoting(): void {
    if (!this.poll || this.updatingStatus) return;

    if (!this.canOpenPoll) {
      this.notificationService.error('You do not have permission to open polls for voting');
      return;
    }

    this.updatingStatus = true;
    this.votingService.updatePollStatus(this.poll.id, 'open').subscribe({
      next: () => {
        this.poll!.status = 'open' as PollStatus;
        this.updatingStatus = false;
        this.notificationService.success('Poll opened for voting successfully');
        this.router.navigate(['/dashboard/polls']);
      },
      error: (err) => {
        const errorMessage = 'Failed to open poll: ' + (err?.error?.error || err?.message || 'Unknown error');
        this.notificationService.error(errorMessage);
        this.updatingStatus = false;
      }
    });
  }

  closePoll(): void {
    if (!this.poll || this.updatingStatus) return;

    this.updatingStatus = true;
    // Use freeze endpoint to compute commitment and distribute shares,
    // backend sets status to 'closed' after distribution
    this.votingService.freezePoll(this.poll.id).subscribe({
      next: () => {
        this.poll!.status = 'closed' as PollStatus;
        this.updatingStatus = false;
        this.notificationService.success('Poll closed and shares distributed successfully');
        this.router.navigate(['/dashboard/polls']);
      },
      error: (err) => {
        const errorMessage = 'Failed to close and distribute shares: ' + (err?.error?.error || err?.message || 'Unknown error');
        this.notificationService.error(errorMessage);
        this.updatingStatus = false;
      }
    });
  }

  freezePoll(): void {
    if (!this.poll || this.updatingStatus) return;

    this.updatingStatus = true;
    this.votingService.freezePoll(this.poll.id).subscribe({
      next: () => {
        this.poll!.status = 'closed' as PollStatus;
        this.updatingStatus = false;
        this.notificationService.success('Poll frozen successfully');
        this.router.navigate(['/dashboard/polls']);
      },
      error: (err) => {
        const errorMessage = 'Failed to freeze poll: ' + (err?.error?.error || err?.message || 'Unknown error');
        this.notificationService.error(errorMessage);
        this.updatingStatus = false;
      }
    });
  }

  getStatusBadgeClass(): string {
    if (!this.poll) return '';
    switch (this.poll.status) {
      case 'draft':
        return 'badge-draft';
      case 'open':
        return 'badge-open';
      case 'closed':
        return 'badge-closed';
      case 'revealed':
        return 'badge-revealed';
      default:
        return '';
    }
  }

  goBack(): void {
    this.router.navigate(['/dashboard/polls']);
  }
}
