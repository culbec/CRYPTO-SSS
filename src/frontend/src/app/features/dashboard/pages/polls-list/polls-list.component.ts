import { Component, OnInit, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { Router, RouterLink, RouterModule } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { Poll, ShareStatus, POLL_STATUS_LABELS } from '../../../../core/models/poll.model';
import { AuthService } from '../../../../core/services/auth.service';
import { forkJoin, of } from 'rxjs';
import { catchError } from 'rxjs/operators';

@Component({
  selector: 'app-polls-list',
  standalone: true,
  imports: [CommonModule, RouterLink, RouterModule],
  templateUrl: './polls-list.component.html',
  styleUrls: ['./polls-list.component.scss']
})
export class PollsListComponent implements OnInit {
  private http = inject(HttpClient);
  private authService = inject(AuthService);
  private router = inject(Router);
  // TODO: move API-related calls to separate service classes
  // TODO: check if status for contributed shares of closed polls is updated correctly on poll cards
  private readonly baseUrl = 'http://localhost:3000/api';

  polls: Poll[] = [];
  filteredPolls: Poll[] = [];
  selectedStatus: string = 'all';
  shareStatuses: Map<string, ShareStatus> = new Map();

  ngOnInit() {
    this.loadPolls();
  }

  loadPolls() {
    this.http.get<any>(`${this.baseUrl}/polls`).subscribe(
      response => {
        this.polls = response.polls || [];
        this.applyFilters();
        this.loadShareStatusesForClosedPolls();
      },
      error => console.error('Failed to load polls', error)
    );
  }

  loadShareStatusesForClosedPolls() {
    const closedPolls = this.polls.filter(p => p.status === 'closed');
    if (closedPolls.length === 0) return;

    const requests = closedPolls.map(poll =>
      this.http.get<ShareStatus>(`${this.baseUrl}/polls/${poll.id}/share-status`).pipe(
        catchError(() => of(null))
      )
    );

    forkJoin(requests).subscribe(statuses => {
      statuses.forEach((status, index) => {
        if (status) {
          this.shareStatuses.set(closedPolls[index].id, status);
        }
      });
    });
  }

  getShareStatus(pollId: string): ShareStatus | undefined {
    return this.shareStatuses.get(pollId);
  }

  filterByStatus(status: string) {
    this.selectedStatus = status;
    this.applyFilters();
  }

  applyFilters() {
    if (this.selectedStatus === 'all') {
      this.filteredPolls = this.polls;
    } else {
      this.filteredPolls = this.polls.filter(p => p.status === this.selectedStatus);
    }
  }

  getStatusLabel(status: string): string {
    return {
      draft: 'Draft',
      open: 'Open',
      closed: 'Closed',
      revealed: 'Revealed'
    }[status] || status;
  }

  getActionLabel(status: string): string {
    const role = this.authService.userRole() ?? 'voter';

    const labelsByRole: Record<string, Record<string, string>> = {
      voter: {
        draft: 'View',
        open: 'Vote Now',
        closed: 'Await Reveal',
        revealed: 'View Results'
      },
      auditor: {
        draft: 'View',
        open: 'View',
        closed: 'Contribute Share',
        revealed: 'View Results'
      },
      official: {
        draft: 'Configure',
        open: 'Manage Poll',
        closed: 'Contribute Share',
        revealed: 'View Results'
      },
      admin: {
        draft: 'Configure',
        open: 'Manage Poll',
        closed: 'Contribute Share',
        revealed: 'View Results'
      }
    };

    const byRole = labelsByRole[role] || labelsByRole['voter'];
    return byRole[status] || 'View';
  }

  getPollRoute(poll: Poll): string {
    const role = this.authService.userRole() ?? 'voter';

    if (poll.status === 'open') {
      return role === 'voter'
        ? `/dashboard/polls/${poll.id}/vote`
        : `/dashboard/polls/${poll.id}`; // auditors and officials can view/manage the poll
    }

    if (poll.status === 'closed') {
      return (role === 'auditor' || role === 'official' || role === 'admin')
        ? `/dashboard/polls/${poll.id}/share`
        : `/dashboard/polls/${poll.id}`; // voters are allowed only to view the poll details
    }

    if (poll.status === 'revealed') {
      return `/dashboard/polls/${poll.id}/results`;
    }

    return `/dashboard/polls/${poll.id}`;
  }

  canCreatePoll(): boolean {
    const role = this.authService.userRole() ?? 'voter';
    return role === 'official' || role === 'admin';
  }

  navigateToCreatePoll() {
    this.router.navigate(['/dashboard/polls/create']);
  }
}
