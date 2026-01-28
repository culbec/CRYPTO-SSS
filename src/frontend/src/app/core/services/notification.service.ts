import { Injectable, inject } from '@angular/core';
import { MatSnackBar, MatSnackBarConfig } from '@angular/material/snack-bar';
import { NotificationToastComponent, NotificationData } from '../../shared/components/notification-toast/notification-toast.component';

@Injectable({
  providedIn: 'root'
})
export class NotificationService {
  private readonly snackBar = inject(MatSnackBar);
  private readonly defaultDuration = 5000;

  success(message: string, duration = this.defaultDuration): void {
    this.showNotification({
      message,
      type: 'success'
    }, duration);
  }

  error(message: string, duration = this.defaultDuration): void {
    this.showNotification({
      message,
      type: 'error'
    }, duration);
  }

  warning(message: string, duration = this.defaultDuration): void {
    this.showNotification({
      message,
      type: 'warning'
    }, duration);
  }

  info(message: string, duration = this.defaultDuration): void {
    this.showNotification({
      message,
      type: 'info'
    }, duration);
  }

  private showNotification(data: NotificationData, duration: number): void {
    this.snackBar.openFromComponent(NotificationToastComponent, {
      data,
      duration,
      horizontalPosition: 'end',
      verticalPosition: 'top',
      panelClass: ['custom-snackbar']
    });
  }

  custom(message: string, config?: MatSnackBarConfig): void {
    this.snackBar.open(message, 'Close', {
      duration: this.defaultDuration,
      horizontalPosition: 'end',
      verticalPosition: 'bottom',
      ...config
    });
  }
}
