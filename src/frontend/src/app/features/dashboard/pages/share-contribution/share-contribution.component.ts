import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { VotingService } from '../../../../core/services/voting.service';
import { AuthService } from '../../../../core/services/auth.service';
import { Poll, ShareStatus, ShareDistribution } from '../../../../core/models/poll.model';

@Component({
  selector: 'app-share-contribution',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './share-contribution.component.html',
  styleUrls: ['./share-contribution.component.scss'] 
})
export class ShareContributionComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private votingService = inject(VotingService);
  private authService = inject(AuthService);
  private router = inject(Router);

  poll: Poll | null = null;
  myShare: ShareDistribution | null = null;
  shareStatus: ShareStatus | null = null;
  shareContributed = false;
  resultsRevealed = false;
  error = '';
  currentUsername = '';

  showSuccessModal = false;
  successMessage = '';
  showErrorModal = false;
  errorModalMessage = '';

  ngOnInit() {
    this.currentUsername = this.authService.user()?.username || '';
    const pollId = this.route.snapshot.paramMap.get('id');
    if (pollId) {
      this.loadPoll(pollId);
      this.loadShareStatus(pollId);
      this.loadMyShare(pollId);
    }
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

  loadShareStatus(pollId: string) {
    this.votingService.getShareStatus(pollId).subscribe({
      next: (status) => this.shareStatus = status,
      error: (err: any) => console.error('Error loading share status', err)
    });
  }

  loadMyShare(pollId: string) {
    this.votingService.getMyShare(pollId).subscribe({
      next: (share) => {
        this.myShare = share;
      },
      error: (err: any) => {
        if (err.status === 404) {
          // User has no share: either poll not closed/distributed yet or user is not a participant
          this.error = 'You do not have a share for this poll. Only auditors and officials receive shares after the poll is closed and distribution runs.';
          console.log('No share assigned to this user for poll:', pollId);
        } else {
          console.error('Error loading share', err);
          this.error = 'Failed to load share: ' + (err?.error?.error || err?.message || 'Unknown error');
        }
      }
    });
  }

  contributeShare() {
    if (!this.myShare || !this.poll) return;

    this.votingService.contributeShare(this.poll.id, this.myShare.share_value).subscribe({
      next: () => {
        this.shareContributed = true;
        // Refresh share status
        this.loadShareStatus(this.poll!.id);
        this.successMessage = 'Share contributed successfully!';
        this.showSuccessModal = true;
      },
      error: (err: any) => {
        console.error('Failed to contribute share', err);
        this.errorModalMessage = 'Error: ' + (err.error?.error || 'Unknown error');
        this.showErrorModal = true;
      }
    });
  }

  revealResults() {
    if (!this.poll) return;

    this.votingService.revealResults(this.poll.id).subscribe({
      next: (response) => {
        // Directly navigate to results page with the revealed data
        this.router.navigate(['/dashboard/polls', this.poll!.id, 'results']);
      },
      error: (err: any) => {
        console.error('Failed to reveal results', err);
        this.errorModalMessage = 'Error: ' + (err.error?.error || 'Unknown error');
        this.showErrorModal = true;
      }
    });
  }

  navigateToResults() {
    if (this.poll) {
      this.router.navigate(['/dashboard/polls', this.poll.id, 'results']);
    }
  }

  hasUserContributed(): boolean {
    if (!this.shareStatus?.contributed_by || this.shareStatus.contributed_by.length === 0) {
      return false;
    }
    return this.shareStatus.contributed_by.includes(this.currentUsername);
  }
}
