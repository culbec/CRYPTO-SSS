import { Component, Inject } from '@angular/core';
import { MAT_SNACK_BAR_DATA, MatSnackBarRef } from '@angular/material/snack-bar';
import { MatIconModule } from '@angular/material/icon';
import { MatButtonModule } from '@angular/material/button';
import { CommonModule } from '@angular/common';

export interface NotificationData {
  message: string;
  type: 'success' | 'error' | 'warning' | 'info';
}

@Component({
  selector: 'app-notification-toast',
  standalone: true,
  imports: [CommonModule, MatIconModule, MatButtonModule],
  template: `
    <div class="notification-toast" [class]="'notification-toast-' + data.type">
      <div class="notification-content">
        <mat-icon class="notification-icon">{{ getIcon() }}</mat-icon>
        <span class="notification-message">{{ data.message }}</span>
      </div>
      <button mat-icon-button (click)="dismiss()" class="notification-close">
        <mat-icon>close</mat-icon>
      </button>
    </div>
  `,
  styleUrls: ['./notification-toast.component.scss']
})
export class NotificationToastComponent {
  constructor(
    @Inject(MAT_SNACK_BAR_DATA) public data: NotificationData,
    private snackBarRef: MatSnackBarRef<NotificationToastComponent>
  ) {}

  getIcon(): string {
    switch (this.data.type) {
      case 'success': return 'check_circle';
      case 'error': return 'error';
      case 'warning': return 'warning';
      case 'info': return 'info';
      default: return 'info';
    }
  }

  dismiss(): void {
    this.snackBarRef.dismiss();
  }
}
