import { Injectable, inject } from '@angular/core';
import { Subject, Observable } from 'rxjs';
import { WebsocketService } from './websocket.service';
import { NotificationService } from './notification.service';
import { AuthService } from './auth.service';

/**
 * Service that handles real-time event notifications and triggers data refetches.
 * Listens to WebSocket events and shows notifications while emitting observables
 * for components to subscribe to and refetch their data.
 */
@Injectable({
  providedIn: 'root'
})
export class PollEventsService {
  private websocket = inject(WebsocketService);
  private notification = inject(NotificationService);
  private auth = inject(AuthService);

  // Observable streams for components to subscribe to
  // Components trigger data refetches when these emit
  private pollCreatedSubject = new Subject<void>();
  private pollStatusChangedSubject = new Subject<{ pollId: string; newStatus: string }>();
  private sharesDistributedSubject = new Subject<string>();
  private shareContributedSubject = new Subject<string>();
  private resultsRevealedSubject = new Subject<string>();

  public pollCreated$ = this.pollCreatedSubject.asObservable();
  public pollStatusChanged$ = this.pollStatusChangedSubject.asObservable();
  public sharesDistributed$ = this.sharesDistributedSubject.asObservable();
  public shareContributed$ = this.shareContributedSubject.asObservable();
  public resultsRevealed$ = this.resultsRevealedSubject.asObservable();

  constructor() {
    this.setupEventListeners();
  }

  private setupEventListeners(): void {
    // Poll created event
    this.websocket.on('poll:created').subscribe((data: any) => {
      console.log('Received poll:created event', data);
      this.notification.info(`New poll created: "${data.pollTitle || 'Poll'}"`);
      this.pollCreatedSubject.next();
    });

    // Poll status changed event
    this.websocket.on('poll:status-changed').subscribe((data: any) => {
      console.log('Received poll:status-changed event', data);
      const pollTitle = data.pollTitle || 'Poll';
      const statusMessages: Record<string, string> = {
        open: `"${pollTitle}" is now open for voting`,
        closed: `"${pollTitle}" has been closed`,
        revealed: `Results for "${pollTitle}" have been revealed!`
      };

      const message = statusMessages[data.newStatus] || `"${pollTitle}" status changed to ${data.newStatus}`;
      this.notification.info(message);
      this.pollStatusChangedSubject.next({
        pollId: data.pollId,
        newStatus: data.newStatus
      });
    });

    // Shares distributed event (only for auditors and officials)
    this.websocket.on('poll:shares-distributed').subscribe((data: any) => {
      console.log('Received poll:shares-distributed event', data);
      
      // Only show share distribution notification to auditors and officials
      const userRole = this.auth.userRole();
      if (userRole === 'auditor' || userRole === 'official' || userRole === 'admin') {
        const pollTitle = data.pollTitle || 'Poll';
        this.notification.info(`Shares distributed for "${pollTitle}" - you can now contribute your share`);
      }
      
      this.sharesDistributedSubject.next(data.pollId);
    });

    // Share contributed event
    this.websocket.on('share:contributed').subscribe((data: any) => {
      console.log('Received share:contributed event', data);
      const pollTitle = data.pollTitle || 'the poll';
      const username = data.username || 'A user';
      this.notification.success(`${username} contributed a share for "${pollTitle}"`);
      this.shareContributedSubject.next(data.pollId);
    });

    // Results revealed event
    this.websocket.on('poll:results-revealed').subscribe((data: any) => {
      console.log('Received poll:results-revealed event', data);
      const pollTitle = data.pollTitle || 'Poll';
      this.notification.success(`Results for "${pollTitle}" have been revealed!`);
      this.resultsRevealedSubject.next(data.pollId);
    });
  }

  onPollStatusChanged(pollId: string): Observable<string> {
    return new Observable(observer => {
      this.pollStatusChanged$.subscribe((data) => {
        if (data.pollId === pollId) {
          observer.next(data.newStatus);
        }
      });
    });
  }

  onShareContributed(pollId: string): Observable<void> {
    return new Observable(observer => {
      this.shareContributed$.subscribe((id) => {
        if (id === pollId) {
          observer.next();
        }
      });
    });
  }

  onSharesDistributed(pollId: string): Observable<void> {
    return new Observable(observer => {
      this.sharesDistributed$.subscribe((id) => {
        if (id === pollId) {
          observer.next();
        }
      });
    });
  }

  onResultsRevealed(pollId: string): Observable<void> {
    return new Observable(observer => {
      this.resultsRevealed$.subscribe((id) => {
        if (id === pollId) {
          observer.next();
        }
      });
    });
  }
}
