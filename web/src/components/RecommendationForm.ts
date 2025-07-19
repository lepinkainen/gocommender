// Recommendation form component for generating recommendations
import { apiClient, ApiError } from '../services/api.js';
import { store } from '../utils/state.js';
import { 
  createElementWithClasses, 
  createElement, 
  createFormField, 
  createButton, 
  createErrorMessage 
} from '../utils/dom.js';
import type { RecommendRequest } from '../types/api.js';

export class RecommendationForm {
  private element: HTMLElement;
  private formElement!: HTMLFormElement;
  private genreInput!: HTMLInputElement;
  private maxResultsInput!: HTMLInputElement;

  constructor() {
    this.element = this.createElement();
  }

  private createElement(): HTMLElement {
    const container = createElementWithClasses('div', 'recommendation-form');
    
    const title = createElementWithClasses('h3', 'form-title', {
      textContent: 'Generate Recommendations'
    });
    container.appendChild(title);

    this.formElement = createElement('form') as HTMLFormElement;
    this.formElement.addEventListener('submit', (e) => this.handleSubmit(e));

    // Genre filter input
    this.genreInput = createElement('input', {
      type: 'text',
      className: 'form-input',
      placeholder: 'Optional: Filter by genre (e.g., "rock", "jazz")'
    }) as HTMLInputElement;
    
    this.genreInput.addEventListener('input', () => {
      store.updateForm({ genre: this.genreInput.value });
    });

    const genreField = createFormField('Genre Filter (Optional)', this.genreInput);
    this.formElement.appendChild(genreField);

    // Max results input
    this.maxResultsInput = createElement('input', {
      type: 'number',
      className: 'form-input',
      min: '1',
      max: '20',
      value: '5'
    }) as HTMLInputElement;
    
    this.maxResultsInput.addEventListener('input', () => {
      const value = parseInt(this.maxResultsInput.value) || 5;
      store.updateForm({ maxResults: Math.min(Math.max(value, 1), 20) });
    });

    const maxResultsField = createFormField('Maximum Results (1-20)', this.maxResultsInput);
    this.formElement.appendChild(maxResultsField);

    // Submit button
    const submitButton = createButton('Generate Recommendations', () => {}, 'primary');
    submitButton.type = 'submit';
    submitButton.id = 'submit-button';
    this.formElement.appendChild(submitButton);

    // Error display area
    const errorContainer = createElementWithClasses('div', 'error-container');
    errorContainer.id = 'form-errors';
    this.formElement.appendChild(errorContainer);

    container.appendChild(this.formElement);

    // Subscribe to state changes
    store.subscribe((state) => {
      this.updateFormState(state.form.genre, state.form.maxResults, state.loading.recommendations, state.errors.recommendations, state.selectedPlaylist);
    });

    return container;
  }

  private updateFormState(
    genre: string, 
    maxResults: number, 
    loading: boolean, 
    error: string | null,
    selectedPlaylist: string | null
  ): void {
    // Update form values if they differ
    if (this.genreInput.value !== genre) {
      this.genreInput.value = genre;
    }
    
    if (parseInt(this.maxResultsInput.value) !== maxResults) {
      this.maxResultsInput.value = maxResults.toString();
    }

    // Update submit button state
    const submitButton = this.element.querySelector('#submit-button') as HTMLButtonElement;
    if (submitButton) {
      submitButton.disabled = loading || !selectedPlaylist;
      submitButton.textContent = loading ? 'Generating...' : 'Generate Recommendations';
    }

    // Update error display
    const errorContainer = this.element.querySelector('#form-errors') as HTMLElement;
    if (errorContainer) {
      errorContainer.innerHTML = '';
      if (error) {
        const errorElement = createErrorMessage(error, true);
        errorContainer.appendChild(errorElement);
      }
    }

    // Show validation message if no playlist selected
    if (!selectedPlaylist && !loading) {
      const validationMessage = createElementWithClasses('p', 'validation-message', {
        textContent: 'Please select a playlist first'
      });
      errorContainer?.appendChild(validationMessage);
    }
  }

  private async handleSubmit(event: Event): Promise<void> {
    event.preventDefault();
    
    const state = store.getState();
    
    if (!state.selectedPlaylist) {
      store.setError('recommendations', 'Please select a playlist first');
      return;
    }

    // Clear previous errors and recommendations
    store.clearError('recommendations');
    store.setState({ recommendations: [] });

    const request: RecommendRequest = {
      playlist_name: state.selectedPlaylist,
      genre: state.form.genre.trim() || undefined,
      max_results: state.form.maxResults
    };

    store.setLoading('recommendations', true);

    try {
      const response = await apiClient.getRecommendations(request);
      
      if (response.status === 'error' || response.error) {
        throw new Error(response.error || 'Failed to generate recommendations');
      }

      store.setState({ 
        recommendations: response.suggestions || [],
      });

      // Scroll to results section
      setTimeout(() => {
        const resultsSection = document.querySelector('.artist-grid');
        if (resultsSection) {
          resultsSection.scrollIntoView({ behavior: 'smooth' });
        }
      }, 100);

    } catch (error) {
      console.error('Recommendation generation failed:', error);
      const errorMessage = error instanceof ApiError ? error.message : 'Failed to generate recommendations';
      store.setError('recommendations', errorMessage);
    } finally {
      store.setLoading('recommendations', false);
    }
  }

  // Public methods
  getElement(): HTMLElement {
    return this.element;
  }

  reset(): void {
    this.formElement.reset();
    store.updateForm({
      genre: '',
      maxResults: 5
    });
    store.clearError('recommendations');
  }

  isValid(): boolean {
    const state = store.getState();
    return !!state.selectedPlaylist && state.form.maxResults >= 1 && state.form.maxResults <= 20;
  }

  getFormData(): RecommendRequest | null {
    const state = store.getState();
    
    if (!state.selectedPlaylist) {
      return null;
    }

    return {
      playlist_name: state.selectedPlaylist,
      genre: state.form.genre.trim() || undefined,
      max_results: state.form.maxResults
    };
  }
}