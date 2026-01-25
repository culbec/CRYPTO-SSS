import { Routes } from '@angular/router';
import { DashboardLayoutComponent } from './pages/dashboard-layout/dashboard-layout.component';
import { PollsListComponent } from './pages/polls-list/polls-list.component';
import { PollDetailComponent } from './pages/poll-detail/poll-detail.component';
import { VotingComponent } from './pages/voting/voting.component';
import { ShareContributionComponent } from './pages/share-contribution/share-contribution.component';
import { PollResultsComponent } from './pages/poll-results/poll-results.component';
import { CreatePollComponent } from './pages/create-poll/create-poll.component';

export const dashboardRoutes: Routes = [
  {
    path: '',
    component: DashboardLayoutComponent,
    children: [
      {
        path: 'polls',
        component: PollsListComponent
      },
      {
        path: 'polls/create',
        component: CreatePollComponent
      },
      {
        path: 'polls/:id/vote',
        component: VotingComponent
      },
      {
        path: 'polls/:id/share',
        component: ShareContributionComponent
      },
      {
        path: 'polls/:id/results',
        component: PollResultsComponent
      },
      {
        path: 'polls/:id',
        component: PollDetailComponent
      },
      {
        path: '',
        redirectTo: 'polls',
        pathMatch: 'full'
      }
    ]
  }
];
