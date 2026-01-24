import { Component } from '@angular/core';
import { CommonModule } from '@angular/common';

@Component({
  selector: 'app-polls',
  standalone: true,
  imports: [CommonModule],
  templateUrl: './polls.component.html',
  styleUrl: './polls.component.scss'
})
export class PollsComponent {}
