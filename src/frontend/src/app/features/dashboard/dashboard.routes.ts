import { Routes } from '@angular/router';
import { DashboardLayoutComponent } from './pages/dashboard-layout/dashboard-layout.component';
import { PollsComponent } from './pages/polls/polls.component';
import { VotingComponent } from './pages/voting/voting.component';
import { ResultsComponent } from './pages/results/results.component';

export const dashboardRoutes: Routes = [
  {
    path: '',
    component: DashboardLayoutComponent,
    children: [
      {
        path: 'polls',
        component: PollsComponent
      },
      {
        path: 'voting',
        component: VotingComponent
      },
      {
        path: 'results',
        component: ResultsComponent
      },
      {
        path: '',
        redirectTo: 'polls',
        pathMatch: 'full'
      }
    ]
  }
];
