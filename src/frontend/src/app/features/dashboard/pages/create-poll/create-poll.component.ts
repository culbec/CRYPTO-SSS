import { Component, inject } from '@angular/core';
import { CommonModule } from '@angular/common';
import { FormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';
import { MatIconModule } from '@angular/material/icon';
import { CreatePollRequest } from '../../../../core/models/poll.model';

@Component({
  selector: 'app-create-poll',
  standalone: true,
  imports: [CommonModule, FormsModule, MatIconModule],
  templateUrl: './create-poll.component.html',
  styleUrls: ['./create-poll.component.scss']
})
export class CreatePollComponent {
  private http = inject(HttpClient);
  private router = inject(Router);
  // TODO: move API-related calls to separate service classes
  private readonly baseUrl = 'http://localhost:3000/api';
  
  poll: CreatePollRequest = this.initialPollState();

  isSubmitting = false;
  errorMessage = '';
  showSuccessModal = false;

  private initialPollState(): CreatePollRequest {
    return {
      title: '',
      description: '',
      options: [
        { id: 'option_1', label: '' },
        { id: 'option_2', label: '' }
      ],
      access_structure_type: 'both',
      min_auditors_required: 1,
      min_officials_required: 2
    };
  }

  addOption() {
    const newId = `option_${this.poll.options.length + 1}`;
    this.poll.options.push({ id: newId, label: '' });
  }

  removeOption(index: number) {
    if (this.poll.options.length > 2) {
      this.poll.options.splice(index, 1);
    }
  }

  createPoll() {
    if (this.isSubmitting) return;

    const hasEmptyOptions = this.poll.options.some(opt => !opt.label.trim());
    if (hasEmptyOptions) {
      this.errorMessage = 'All options must have a label';
      return;
    }

    if (this.poll.access_structure_type === 'auditors_only' || this.poll.access_structure_type === 'both') {
      if (this.poll.min_auditors_required < 1) {
        this.errorMessage = 'Minimum auditors required must be at least 1';
        return;
      }
    }
    if (this.poll.access_structure_type === 'officials_only' || this.poll.access_structure_type === 'both') {
      if (this.poll.min_officials_required < 1) {
        this.errorMessage = 'Minimum officials required must be at least 1';
        return;
      }
    }

    this.isSubmitting = true;
    this.errorMessage = '';

    this.http.post(`${this.baseUrl}/polls`, this.poll).subscribe({
      next: (response: any) => {
        console.log('Poll created:', response);
        this.showSuccessModal = true;
        this.isSubmitting = false;
      },
      error: (error) => {
        console.error('Failed to create poll', error);
        this.errorMessage = error.error?.error || 'Failed to create poll. Please try again.';
        this.isSubmitting = false;
      }
    });
  }

  createAnotherPoll() {
    this.poll = this.initialPollState();
    this.errorMessage = '';
    this.showSuccessModal = false;
  }

  closeSuccessModal() {
    this.showSuccessModal = false;
  }

  goToPolls() {
    this.showSuccessModal = false;
    this.router.navigate(['/dashboard/polls']);
  }

  cancel() {
    this.router.navigate(['/dashboard/polls']);
  }
}
