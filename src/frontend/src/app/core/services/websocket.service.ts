import { Injectable, signal } from '@angular/core';
import { Observable, Subject } from 'rxjs';

interface WebSocketMessage {
  event: string;
  data: any;
}

/**
 * Service that manages WebSocket connection for real-time updates using native WebSocket.
 */
@Injectable({
  providedIn: 'root'
})
export class WebsocketService {
  private readonly wsUrl = 'ws://localhost:3000/ws';

  private ws: WebSocket | null = null;
  private isConnected = signal<boolean>(false);
  private eventSubjects = new Map<string, Subject<any>>();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectDelay = 1000;

  constructor() {
    this.initializeWebSocket();
  }

  private initializeWebSocket(): void {
    try {
      this.ws = new WebSocket(this.wsUrl);
      this.setupConnectionHandlers();
    } catch (err) {
      console.error('Could not initialize WebSocket:', err);
      this.scheduleReconnect();
    }
  }

  private setupConnectionHandlers(): void {
    if (!this.ws) return;

    this.ws.onopen = () => {
      console.log('WebSocket connected to', this.wsUrl);
      this.isConnected.set(true);
      this.reconnectAttempts = 0;
    };

    this.ws.onclose = () => {
      console.log('WebSocket disconnected');
      this.isConnected.set(false);
      this.scheduleReconnect();
    };

    this.ws.onerror = (error) => {
      console.error('WebSocket error:', error);
    };

    this.ws.onmessage = (event) => {
      try {
        const message: WebSocketMessage = JSON.parse(event.data);
        console.log('WebSocket message received:', message);

        // Emit to the appropriate subject
        const subject = this.eventSubjects.get(message.event);
        if (subject) {
          console.log('Received', message.event, 'event', message.data);
          subject.next(message.data);
        }
      } catch (err) {
        console.error('Failed to parse WebSocket message:', err);
      }
    };
  }

  private scheduleReconnect(): void {
    if (this.reconnectAttempts >= this.maxReconnectAttempts) {
      console.error('Max WebSocket reconnect attempts reached');
      return;
    }

    this.reconnectAttempts++;
    const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);

    console.log(`Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
    setTimeout(() => {
      this.initializeWebSocket();
    }, delay);
  }

  on(event: string): Observable<any> {
    if (!this.eventSubjects.has(event)) {
      this.eventSubjects.set(event, new Subject<any>());
    }
    return this.eventSubjects.get(event)!.asObservable();
  }

  emit(event: string, data: any): void {
    if (this.ws && this.ws.readyState === WebSocket.OPEN) {
      const message: WebSocketMessage = { event, data };
      this.ws.send(JSON.stringify(message));
    } else {
      console.warn('WebSocket not connected, cannot emit event:', event);
    }
  }

  getIsConnected(): boolean {
    return this.isConnected();
  }

  disconnect(): void {
    if (this.ws) {
      this.ws.close();
      this.ws = null;
    }
  }
}
