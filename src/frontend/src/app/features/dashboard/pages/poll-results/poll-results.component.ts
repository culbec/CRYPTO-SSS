import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { ActivatedRoute, Router } from '@angular/router';
import { VotingService } from '../../../../core/services/voting.service';
import { Poll, PollResultResponse } from '../../../../core/models/poll.model';

@Component({
  selector: 'app-poll-results',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './poll-results.component.html',
  styleUrls: ['./poll-results.component.scss'],
})
export class PollResultsComponent implements OnInit {
  private route = inject(ActivatedRoute);
  private votingService = inject(VotingService);
  private router = inject(Router);

  poll: Poll | null = null;
  pollResults: PollResultResponse | null = null;
  loading = true;
  error: string | null = null;

  ngOnInit() {
    const pollId = this.route.snapshot.paramMap.get('id');
    if (pollId) {
      this.loadPollAndResults(pollId);
    }
  }

  loadPollAndResults(pollId: string) {
    // First load the poll details
    this.votingService.getPollById(pollId).subscribe({
      next: (poll) => {
        this.poll = poll;
        
        // If poll is revealed, try to fetch the results
        if (poll.status === 'revealed') {
          this.votingService.getPollResults(pollId).subscribe({
            next: (results) => {
              this.pollResults = results;
              this.loading = false;
            },
            error: (err) => {
              console.error('Failed to load results:', err);
              // Fall back to showing poll details even if results endpoint fails
              this.loading = false;
            }
          });
        } else {
          this.error = 'Poll results are not yet available. Poll status: ' + poll.status;
          this.loading = false;
        }
      },
      error: (err) => {
        console.error('Failed to load poll', err);
        this.error = 'Failed to load poll: ' + (err?.error?.error || err?.message || 'Unknown error');
        this.loading = false;
      }
    });
  }

  getVoteCount(optionId: string): number {
    if (!this.pollResults || !this.pollResults.results) return 0;
    return this.pollResults.results[optionId] || 0;
  }

  getPercentage(optionId: string): number {
    if (!this.pollResults || this.pollResults.total_votes === 0) return 0;
    return (this.getVoteCount(optionId) / this.pollResults.total_votes) * 100;
  }

  getMaxVotes(): number {
    if (!this.poll || !this.pollResults) return 0;
    return Math.max(...this.poll.options.map(o => this.getVoteCount(o.id)));
  }

  getWinnerOption() {
    if (!this.poll || !this.pollResults) return null;
    return this.poll.options.find(o => this.getVoteCount(o.id) === this.getMaxVotes());
  }

  getWinnerPercentage(): number {
    if (!this.pollResults || this.pollResults.total_votes === 0) return 0;
    return (this.getMaxVotes() / this.pollResults.total_votes) * 100;
  }

  isTie(): boolean {
    if (!this.poll || !this.pollResults) return false;
    const maxVotes = this.getMaxVotes();
    return this.poll.options.filter(o => this.getVoteCount(o.id) === maxVotes).length > 1;
  }

  formatDate(dateString: string): string {
    return new Date(dateString).toLocaleString();
  }

  goBack() {
    this.router.navigate(['/dashboard/polls']);
  }
}
