// DOM utility functions for component creation

// Create element with attributes and content
export function createElement<T extends keyof HTMLElementTagNameMap>(
  tag: T,
  attributes: Partial<HTMLElementTagNameMap[T]> = {},
  ...children: (string | Node)[]
): HTMLElementTagNameMap[T] {
  const element = document.createElement(tag);
  
  // Set attributes
  Object.assign(element, attributes);
  
  // Append children
  children.forEach(child => {
    if (typeof child === 'string') {
      element.appendChild(document.createTextNode(child));
    } else {
      element.appendChild(child);
    }
  });
  
  return element;
}

// Create element with CSS classes
export function createElementWithClasses<T extends keyof HTMLElementTagNameMap>(
  tag: T,
  classes: string | string[],
  attributes: Partial<HTMLElementTagNameMap[T]> = {},
  ...children: (string | Node)[]
): HTMLElementTagNameMap[T] {
  const element = createElement(tag, attributes, ...children);
  
  if (Array.isArray(classes)) {
    element.classList.add(...classes);
  } else {
    element.classList.add(...classes.split(' ').filter(Boolean));
  }
  
  return element;
}

// Create a loading spinner element
export function createLoadingSpinner(size: 'small' | 'medium' | 'large' = 'medium'): HTMLElement {
  return createElementWithClasses('div', [`loading-spinner`, `loading-spinner--${size}`], {
    innerHTML: '⟳'
  });
}

// Create an error message element
export function createErrorMessage(message: string, dismissible: boolean = true): HTMLElement {
  const errorDiv = createElementWithClasses('div', 'error-message');
  
  const messageSpan = createElement('span', { textContent: message });
  errorDiv.appendChild(messageSpan);
  
  if (dismissible) {
    const dismissButton = createElementWithClasses('button', 'error-dismiss', {
      textContent: '×',
      title: 'Dismiss error'
    });
    
    dismissButton.addEventListener('click', () => {
      errorDiv.remove();
    });
    
    errorDiv.appendChild(dismissButton);
  }
  
  return errorDiv;
}

// Create a badge element
export function createBadge(text: string, variant: 'primary' | 'secondary' | 'success' | 'warning' | 'error' = 'primary'): HTMLElement {
  return createElementWithClasses('span', [`badge`, `badge--${variant}`], {
    textContent: text
  });
}

// Create a card element
export function createCard(title?: string, ...content: Node[]): HTMLElement {
  const card = createElementWithClasses('div', 'card');
  
  if (title) {
    const header = createElementWithClasses('div', 'card-header');
    const titleElement = createElementWithClasses('h3', 'card-title', { textContent: title });
    header.appendChild(titleElement);
    card.appendChild(header);
  }
  
  if (content.length > 0) {
    const body = createElementWithClasses('div', 'card-body');
    content.forEach(node => body.appendChild(node));
    card.appendChild(body);
  }
  
  return card;
}

// Create a button with loading state support
export function createButton(
  text: string,
  onClick: () => void | Promise<void>,
  variant: 'primary' | 'secondary' | 'outline' = 'primary'
): HTMLButtonElement {
  const button = createElementWithClasses('button', [`btn`, `btn--${variant}`], {
    textContent: text
  }) as HTMLButtonElement;
  
  button.addEventListener('click', async () => {
    const originalText = button.textContent;
    button.disabled = true;
    button.innerHTML = '';
    button.appendChild(createLoadingSpinner('small'));
    
    try {
      await onClick();
    } finally {
      button.disabled = false;
      button.innerHTML = '';
      button.textContent = originalText;
    }
  });
  
  return button;
}

// Create a form field with label
export function createFormField(
  label: string,
  input: HTMLInputElement | HTMLSelectElement | HTMLTextAreaElement,
  error?: string
): HTMLElement {
  const field = createElementWithClasses('div', 'form-field');
  
  const labelElement = createElementWithClasses('label', 'form-label', {
    textContent: label
  });
  
  field.appendChild(labelElement);
  field.appendChild(input);
  
  if (error) {
    const errorElement = createElementWithClasses('div', 'form-error', {
      textContent: error
    });
    field.appendChild(errorElement);
    field.classList.add('form-field--error');
  }
  
  return field;
}

// Create a select element with options
export function createSelect(
  options: { value: string; text: string; selected?: boolean }[],
  placeholder?: string
): HTMLSelectElement {
  const select = createElement('select', { className: 'form-input' }) as HTMLSelectElement;
  
  if (placeholder) {
    const placeholderOption = createElement('option', {
      value: '',
      textContent: placeholder,
      disabled: true,
      selected: true
    });
    select.appendChild(placeholderOption);
  }
  
  options.forEach(option => {
    const optionElement = createElement('option', {
      value: option.value,
      textContent: option.text,
      selected: option.selected
    });
    select.appendChild(optionElement);
  });
  
  return select;
}

// Format time duration
export function formatDuration(milliseconds: number): string {
  const seconds = Math.floor(milliseconds / 1000);
  const minutes = Math.floor(seconds / 60);
  const hours = Math.floor(minutes / 60);
  
  if (hours > 0) {
    return `${hours}:${(minutes % 60).toString().padStart(2, '0')}:${(seconds % 60).toString().padStart(2, '0')}`;
  } else {
    return `${minutes}:${(seconds % 60).toString().padStart(2, '0')}`;
  }
}

// Format date
export function formatDate(dateString: string): string {
  const date = new Date(dateString);
  return date.toLocaleDateString() + ' ' + date.toLocaleTimeString();
}

// Truncate text with ellipsis
export function truncateText(text: string, maxLength: number): string {
  if (text.length <= maxLength) return text;
  return text.substring(0, maxLength - 3) + '...';
}

// Escape HTML
export function escapeHtml(text: string): string {
  const div = document.createElement('div');
  div.textContent = text;
  return div.innerHTML;
}

// Check if element is in viewport
export function isInViewport(element: HTMLElement): boolean {
  const rect = element.getBoundingClientRect();
  return (
    rect.top >= 0 &&
    rect.left >= 0 &&
    rect.bottom <= (window.innerHeight || document.documentElement.clientHeight) &&
    rect.right <= (window.innerWidth || document.documentElement.clientWidth)
  );
}