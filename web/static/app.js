class MailboxZero {
    constructor() {
        this.emails = [];
        this.similarEmails = [];
        this.selectedEmailId = null;
        this.selectedSimilarEmails = new Set();
        this.inboxSortBy = 'date'; // Default sort by date (newest first)
        this.similarSortBy = 'date'; // Default sort by date (newest first)
        this.totalInboxCount = 0; // Track total count from server
        
        // Pagination state
        this.currentPage = 1;
        this.perPage = 100;
        this.totalPages = 1;
        
        this.initializeElements();
        this.attachEventListeners();
        this.initializeTitles();
        this.loadEmails();
    }

    initializeElements() {
        this.similaritySlider = document.getElementById('similarity-slider');
        this.similarityValue = document.getElementById('similarity-value');
        this.refreshBtn = document.getElementById('refresh-btn');
        this.findSimilarBtn = document.getElementById('find-similar-btn');
        this.clearResultsBtn = document.getElementById('clear-results-btn');
        this.selectAllCheckbox = document.getElementById('select-all-checkbox');
        this.archiveBtn = document.getElementById('archive-btn');
        this.inboxList = document.getElementById('inbox-list');
        this.similarList = document.getElementById('similar-list');
        
        // Title elements
        this.inboxTitle = document.getElementById('inbox-title');
        this.similarTitle = document.getElementById('similar-title');
        
        // Sort controls
        this.inboxSortSelect = document.getElementById('inbox-sort');
        this.similarSortSelect = document.getElementById('similar-sort');
        
        // Modal elements
        this.archiveModal = document.getElementById('archive-modal');
        this.modalOverlay = document.getElementById('modal-overlay');
        this.confirmArchiveBtn = document.getElementById('confirm-archive-btn');
        this.cancelArchiveBtn = document.getElementById('cancel-archive-btn');
        this.archiveCount = document.getElementById('archive-count');
        
        // Preview popup elements
        this.previewPopup = document.getElementById('email-preview-popup');
        this.previewSubject = document.getElementById('preview-subject');
        this.previewFrom = document.getElementById('preview-from');
        this.previewDate = document.getElementById('preview-date');
        this.previewBody = document.getElementById('preview-body');
        
        // Pagination elements
        this.perPageSelect = document.getElementById('per-page-select');
        this.showingStart = document.getElementById('showing-start');
        this.showingEnd = document.getElementById('showing-end');
        this.totalEmailsSpan = document.getElementById('total-emails');
        this.currentPageSpan = document.getElementById('current-page');
        this.totalPagesSpan = document.getElementById('total-pages');
        this.prevPageBtn = document.getElementById('prev-page-btn');
        this.nextPageBtn = document.getElementById('next-page-btn');
        
        // Preview toggle checkbox
        this.previewToggleCheckbox = document.getElementById('preview-toggle-checkbox');
        
        // Preview state
        this.previewTimeout = null;
        this.hidePreviewTimeout = null;
        this.currentPreviewEmail = null;
        this.isMouseOverPreview = false;
        this.previewsEnabled = true;
    }

    initializeTitles() {
        // Initialize titles with zero counts
        this.inboxTitle.textContent = 'Inbox (0)';
        this.similarTitle.textContent = 'Similar Emails (0)';
    }

    attachEventListeners() {
        this.similaritySlider.addEventListener('input', (e) => {
            this.similarityValue.textContent = e.target.value + '%';
        });

        this.refreshBtn.addEventListener('click', () => this.loadEmails());
        this.findSimilarBtn.addEventListener('click', () => this.findSimilarEmails());
        this.clearResultsBtn.addEventListener('click', () => this.clearResults());
        
        this.selectAllCheckbox.addEventListener('change', (e) => {
            this.toggleAllSimilarEmails(e.target.checked);
        });
        
        // Sort event listeners
        this.inboxSortSelect.addEventListener('change', (e) => {
            this.inboxSortBy = e.target.value;
            this.renderEmails(this.emails, this.inboxList, false);
        });
        
        this.similarSortSelect.addEventListener('change', (e) => {
            this.similarSortBy = e.target.value;
            this.renderEmails(this.similarEmails, this.similarList, true);
        });
        
        this.archiveBtn.addEventListener('click', () => this.showArchiveModal());
        this.confirmArchiveBtn.addEventListener('click', () => this.archiveEmails());
        this.cancelArchiveBtn.addEventListener('click', () => this.hideArchiveModal());
        this.modalOverlay.addEventListener('click', () => this.hideArchiveModal());
        
        // Preview toggle event listener
        this.previewToggleCheckbox.addEventListener('change', (e) => {
            this.previewsEnabled = e.target.checked;
            if (!this.previewsEnabled) {
                this.hideEmailPreview();
            }
        });
        
        // Hide preview when clicking elsewhere or pressing keys
        document.addEventListener('click', (e) => {
            // Don't hide if clicking inside the preview popup
            if (!e.target.closest('.email-preview-popup')) {
                this.hideEmailPreview();
            }
        });
        document.addEventListener('keydown', (e) => {
            // Hide on escape key
            if (e.key === 'Escape') {
                this.hideEmailPreview();
            }
        });
        
        // Add mouse tracking for preview popup
        this.previewPopup.addEventListener('mouseenter', () => {
            this.isMouseOverPreview = true;
            // Clear any pending hide timeout
            if (this.hidePreviewTimeout) {
                clearTimeout(this.hidePreviewTimeout);
                this.hidePreviewTimeout = null;
            }
        });
        
        this.previewPopup.addEventListener('mouseleave', () => {
            this.isMouseOverPreview = false;
            // Give a small delay before hiding
            this.hidePreviewTimeout = setTimeout(() => {
                this.hideEmailPreview();
            }, 100);
        });
        
        // Pagination event listeners
        this.perPageSelect.addEventListener('change', (e) => {
            this.perPage = parseInt(e.target.value);
            this.currentPage = 1; // Reset to first page
            this.loadEmails();
        });
        
        this.prevPageBtn.addEventListener('click', () => {
            if (this.currentPage > 1) {
                this.currentPage--;
                this.loadEmails();
            }
        });
        
        this.nextPageBtn.addEventListener('click', () => {
            if (this.currentPage < this.totalPages) {
                this.currentPage++;
                this.loadEmails();
            }
        });
    }

    async loadEmails() {
        try {
            this.showLoading(this.inboxList, 'Loading emails...');
            
            const offset = (this.currentPage - 1) * this.perPage;
            const url = `/api/emails?limit=${this.perPage}&offset=${offset}`;
            
            const response = await fetch(url);
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const inboxInfo = await response.json();
            this.emails = inboxInfo.emails;
            this.totalInboxCount = inboxInfo.totalCount;
            
            // Calculate pagination
            this.totalPages = Math.ceil(this.totalInboxCount / this.perPage);
            
            this.renderEmails(this.emails, this.inboxList, false);
            this.updateTitles();
            this.updatePaginationInfo();
        } catch (error) {
            console.error('Error loading emails:', error);
            this.showError(this.inboxList, 'Failed to load emails. Please check your configuration.');
        }
    }

    async findSimilarEmails() {
        try {
            this.showLoading(this.similarList, 'Finding similar emails...');
            
            const similarityThreshold = parseFloat(this.similaritySlider.value);
            const requestBody = {
                similarityThreshold: similarityThreshold
            };
            
            if (this.selectedEmailId) {
                requestBody.emailId = this.selectedEmailId;
            }
            
            const response = await fetch('/api/similar', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify(requestBody)
            });
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            this.similarEmails = await response.json();
            this.selectedSimilarEmails.clear();
            
            if (this.similarEmails.length === 0) {
                this.showEmpty(this.similarList, 'No similar emails found with the current similarity threshold.');
            } else {
                this.similarEmails.forEach(email => this.selectedSimilarEmails.add(email.id));
                this.renderEmails(this.similarEmails, this.similarList, true);
                this.selectAllCheckbox.checked = true;
            }
            
            this.updateControls();
        } catch (error) {
            console.error('Error finding similar emails:', error);
            this.showError(this.similarList, 'Failed to find similar emails.');
        }
    }

    async archiveEmails() {
        try {
            const emailIds = Array.from(this.selectedSimilarEmails);
            
            const response = await fetch('/api/archive', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                },
                body: JSON.stringify({ emailIds })
            });
            
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const result = await response.json();
            this.hideArchiveModal();
            
            if (result.dryRun) {
                alert(`Dry run completed: Would have archived ${emailIds.length} emails.`);
            } else {
                alert(`Successfully archived ${emailIds.length} emails.`);
                this.loadEmails(); // Refresh inbox
                this.clearResults(); // Clear similar emails
            }
        } catch (error) {
            console.error('Error archiving emails:', error);
            alert('Failed to archive emails.');
            this.hideArchiveModal();
        }
    }

    async clearResults() {
        try {
            await fetch('/api/clear', { method: 'POST' });
            this.similarEmails = [];
            this.selectedSimilarEmails.clear();
            this.showEmpty(this.similarList, 
                'Click "Find Similar Emails" to find duplicate or similar messages<br>' +
                'Or select a specific email from the left and then click "Find Similar Emails"'
            );
            this.updateTitles();
            this.updateControls();
        } catch (error) {
            console.error('Error clearing results:', error);
        }
    }

    sortEmails(emails, sortBy) {
        const sortedEmails = [...emails];
        
        switch (sortBy) {
            case 'date':
                // Default: newest first (descending)
                sortedEmails.sort((a, b) => {
                    const dateA = new Date(a.receivedAt || 0);
                    const dateB = new Date(b.receivedAt || 0);
                    return dateB - dateA;
                });
                break;
            case 'date-asc':
                // Oldest first (ascending)
                sortedEmails.sort((a, b) => {
                    const dateA = new Date(a.receivedAt || 0);
                    const dateB = new Date(b.receivedAt || 0);
                    return dateA - dateB;
                });
                break;
            case 'subject':
                // Subject A-Z (ascending)
                sortedEmails.sort((a, b) => {
                    const subjectA = (a.subject || '').toLowerCase();
                    const subjectB = (b.subject || '').toLowerCase();
                    return subjectA.localeCompare(subjectB);
                });
                break;
            case 'subject-desc':
                // Subject Z-A (descending)
                sortedEmails.sort((a, b) => {
                    const subjectA = (a.subject || '').toLowerCase();
                    const subjectB = (b.subject || '').toLowerCase();
                    return subjectB.localeCompare(subjectA);
                });
                break;
            case 'sender':
                // Sender A-Z (ascending)
                sortedEmails.sort((a, b) => {
                    const senderA = this.getSenderName(a).toLowerCase();
                    const senderB = this.getSenderName(b).toLowerCase();
                    return senderA.localeCompare(senderB);
                });
                break;
            case 'sender-desc':
                // Sender Z-A (descending)
                sortedEmails.sort((a, b) => {
                    const senderA = this.getSenderName(a).toLowerCase();
                    const senderB = this.getSenderName(b).toLowerCase();
                    return senderB.localeCompare(senderA);
                });
                break;
            default:
                // Default to date descending
                sortedEmails.sort((a, b) => {
                    const dateA = new Date(a.receivedAt || 0);
                    const dateB = new Date(b.receivedAt || 0);
                    return dateB - dateA;
                });
        }
        
        return sortedEmails;
    }

    getSenderName(email) {
        if (email.from && email.from.length > 0) {
            return email.from[0].name || email.from[0].email || 'Unknown sender';
        }
        return 'Unknown sender';
    }

    renderEmails(emails, container, withCheckboxes) {
        // Determine which sort to use based on which container we're rendering to
        const sortBy = container === this.inboxList ? this.inboxSortBy : this.similarSortBy;
        const sortedEmails = this.sortEmails(emails, sortBy);
        
        const emailsHtml = sortedEmails.map(email => {
            const fromName = this.getSenderName(email);
            
            const date = email.receivedAt ? 
                new Date(email.receivedAt).toLocaleDateString() : '';
                
            // Only apply selection highlight to inbox emails, not similar emails
            const isSelected = !withCheckboxes && email.id === this.selectedEmailId;
            const isChecked = withCheckboxes && this.selectedSimilarEmails.has(email.id);
            
            return `
                <div class="email-item ${isSelected ? 'selected' : ''}" 
                     data-email-id="${email.id}"
                     data-with-checkbox="${withCheckboxes}">
                    ${withCheckboxes ? `
                        <input type="checkbox" class="email-checkbox" 
                               ${isChecked ? 'checked' : ''}>
                    ` : ''}
                    <div class="email-content">
                        <div class="email-subject">${this.escapeHtml(email.subject || '(No subject)')}</div>
                        <div class="email-from">${this.escapeHtml(fromName)}</div>
                        <div class="email-preview">${this.escapeHtml(email.preview || '')}</div>
                    </div>
                    <div class="email-date">${date}</div>
                </div>
            `;
        }).join('');
        
        container.innerHTML = emailsHtml;
        
        // Attach click and hover listeners
        container.querySelectorAll('.email-item').forEach(item => {
            const emailId = item.dataset.emailId;
            const email = sortedEmails.find(e => e.id === emailId);
            
            // Add preview hover functionality
            item.addEventListener('mouseenter', (e) => {
                // Cancel any pending hide
                if (this.hidePreviewTimeout) {
                    clearTimeout(this.hidePreviewTimeout);
                    this.hidePreviewTimeout = null;
                }
                this.showEmailPreview(e, email);
            });
            
            item.addEventListener('mouseleave', (e) => {
                // Don't hide immediately if mouse is moving to the preview popup
                this.hidePreviewTimeout = setTimeout(() => {
                    if (!this.isMouseOverPreview) {
                        this.hideEmailPreview();
                    }
                }, 150);
            });
            
            item.addEventListener('mousemove', (e) => {
                this.updatePreviewPosition(e);
            });
            
            if (withCheckboxes) {
                // For similar emails, handle checkbox clicks
                const checkbox = item.querySelector('.email-checkbox');
                checkbox.addEventListener('change', (e) => {
                    e.stopPropagation();
                    const emailId = item.dataset.emailId;
                    if (e.target.checked) {
                        this.selectedSimilarEmails.add(emailId);
                    } else {
                        this.selectedSimilarEmails.delete(emailId);
                    }
                    this.updateSelectAllCheckbox();
                    this.updateControls();
                });
                
                // Also allow clicking the item to toggle checkbox
                item.addEventListener('click', (e) => {
                    if (e.target !== checkbox) {
                        checkbox.checked = !checkbox.checked;
                        checkbox.dispatchEvent(new Event('change', { bubbles: true }));
                    }
                });
            } else {
                // For inbox emails, handle selection
                item.addEventListener('click', () => {
                    this.selectEmail(item.dataset.emailId);
                });
            }
        });
    }

    selectEmail(emailId) {
        // If clicking the same email that's already selected, unselect it
        if (this.selectedEmailId === emailId) {
            const selectedItem = this.inboxList.querySelector(`[data-email-id="${emailId}"]`);
            if (selectedItem) {
                selectedItem.classList.remove('selected');
            }
            this.selectedEmailId = null;
            return;
        }
        
        // Remove selection from all items
        this.inboxList.querySelectorAll('.email-item').forEach(item => {
            item.classList.remove('selected');
        });
        
        // Add selection to clicked item
        const selectedItem = this.inboxList.querySelector(`[data-email-id="${emailId}"]`);
        if (selectedItem) {
            selectedItem.classList.add('selected');
            this.selectedEmailId = emailId;
        }
    }

    toggleAllSimilarEmails(checked) {
        if (checked) {
            this.similarEmails.forEach(email => {
                this.selectedSimilarEmails.add(email.id);
            });
        } else {
            this.selectedSimilarEmails.clear();
        }
        
        // Update checkboxes
        this.similarList.querySelectorAll('.email-checkbox').forEach(checkbox => {
            checkbox.checked = checked;
        });
        
        this.updateControls();
    }

    updateSelectAllCheckbox() {
        const totalEmails = this.similarEmails.length;
        const selectedCount = this.selectedSimilarEmails.size;
        
        if (selectedCount === 0) {
            this.selectAllCheckbox.checked = false;
            this.selectAllCheckbox.indeterminate = false;
        } else if (selectedCount === totalEmails) {
            this.selectAllCheckbox.checked = true;
            this.selectAllCheckbox.indeterminate = false;
        } else {
            this.selectAllCheckbox.checked = false;
            this.selectAllCheckbox.indeterminate = true;
        }
    }

    updateControls() {
        const hasResults = this.similarEmails.length > 0;
        const hasSelected = this.selectedSimilarEmails.size > 0;
        
        this.clearResultsBtn.disabled = !hasResults;
        this.archiveBtn.disabled = !hasSelected;
        
        this.updateTitles();
    }

    updateTitles() {
        // Update inbox title with total count from server, not just fetched count
        this.inboxTitle.textContent = `Inbox (${this.totalInboxCount})`;
        
        // Update similar emails title with count
        const similarCount = this.similarEmails.length;
        this.similarTitle.textContent = `Similar Emails (${similarCount})`;
    }

    updatePaginationInfo() {
        const start = this.emails.length > 0 ? ((this.currentPage - 1) * this.perPage) + 1 : 0;
        const end = Math.min(start + this.emails.length - 1, this.totalInboxCount);
        
        this.showingStart.textContent = start;
        this.showingEnd.textContent = end;
        this.totalEmailsSpan.textContent = this.totalInboxCount;
        this.currentPageSpan.textContent = this.currentPage;
        this.totalPagesSpan.textContent = this.totalPages;
        
        // Update navigation buttons
        this.prevPageBtn.disabled = this.currentPage <= 1;
        this.nextPageBtn.disabled = this.currentPage >= this.totalPages;
    }

    showArchiveModal() {
        const count = this.selectedSimilarEmails.size;
        this.archiveCount.textContent = count;
        this.archiveModal.style.display = 'block';
        this.modalOverlay.style.display = 'block';
    }

    hideArchiveModal() {
        this.archiveModal.style.display = 'none';
        this.modalOverlay.style.display = 'none';
    }

    showLoading(container, message) {
        container.innerHTML = `<div class="loading">${message}</div>`;
    }

    showError(container, message) {
        container.innerHTML = `<div class="empty-state"><p style="color: #e74c3c;">${message}</p></div>`;
    }

    showEmpty(container, message) {
        container.innerHTML = `<div class="empty-state"><p>${message}</p></div>`;
    }

    escapeHtml(text) {
        const div = document.createElement('div');
        div.textContent = text;
        return div.innerHTML;
    }

    // Email Preview Methods
    showEmailPreview(event, email) {
        // Don't show preview if previews are disabled
        if (!this.previewsEnabled) {
            return;
        }
        
        // Clear any existing timeouts
        if (this.previewTimeout) {
            clearTimeout(this.previewTimeout);
        }
        if (this.hidePreviewTimeout) {
            clearTimeout(this.hidePreviewTimeout);
            this.hidePreviewTimeout = null;
        }

        // Set a delay before showing preview
        this.previewTimeout = setTimeout(() => {
            this.displayPreview(event, email);
        }, 300); // 300ms delay
    }

    hideEmailPreview() {
        if (this.previewTimeout) {
            clearTimeout(this.previewTimeout);
            this.previewTimeout = null;
        }
        if (this.hidePreviewTimeout) {
            clearTimeout(this.hidePreviewTimeout);
            this.hidePreviewTimeout = null;
        }
        this.previewPopup.style.display = 'none';
        this.currentPreviewEmail = null;
        this.isMouseOverPreview = false;
    }

    displayPreview(event, email) {
        this.currentPreviewEmail = email;
        
        // Set basic info immediately
        this.previewSubject.textContent = email.subject || '(No subject)';
        this.previewFrom.textContent = this.getSenderName(email);
        
        const date = email.receivedAt ? 
            new Date(email.receivedAt).toLocaleDateString('en-US', {
                year: 'numeric',
                month: 'short',
                day: 'numeric',
                hour: '2-digit',
                minute: '2-digit'
            }) : '';
        this.previewDate.textContent = date;

        // Show loading state for body
        this.previewBody.innerHTML = '<div class="preview-loading">Loading preview...</div>';
        
        // Position and show popup
        this.positionPreviewPopup(event);
        this.previewPopup.style.display = 'block';
        
        // Load email body content
        this.loadPreviewBody(email);
    }

    loadPreviewBody(email) {
        let bodyContent = '';
        
        // First try to find text body content using textBody structure
        if (email.bodyValues && email.textBody && email.textBody.length > 0) {
            // Look for text parts first (preferred for preview)
            for (const part of email.textBody) {
                if (part.partId && email.bodyValues[part.partId]) {
                    const bodyValue = email.bodyValues[part.partId];
                    if (bodyValue.value && bodyValue.value.trim()) {
                        bodyContent = bodyValue.value;
                        break;
                    }
                }
            }
        }
        
        // If no text body, try HTML body but strip HTML tags
        if (!bodyContent && email.bodyValues && email.htmlBody && email.htmlBody.length > 0) {
            for (const part of email.htmlBody) {
                if (part.partId && email.bodyValues[part.partId]) {
                    const bodyValue = email.bodyValues[part.partId];
                    if (bodyValue.value && bodyValue.value.trim()) {
                        // Strip HTML tags for preview
                        const tempDiv = document.createElement('div');
                        tempDiv.innerHTML = bodyValue.value;
                        bodyContent = tempDiv.textContent || tempDiv.innerText || '';
                        break;
                    }
                }
            }
        }
        
        // Fallback: try any bodyValues content
        if (!bodyContent && email.bodyValues) {
            for (const partId in email.bodyValues) {
                const bodyValue = email.bodyValues[partId];
                if (bodyValue.value && bodyValue.value.trim()) {
                    // If it looks like HTML, strip tags
                    if (bodyValue.value.includes('<') && bodyValue.value.includes('>')) {
                        const tempDiv = document.createElement('div');
                        tempDiv.innerHTML = bodyValue.value;
                        bodyContent = tempDiv.textContent || tempDiv.innerText || '';
                    } else {
                        bodyContent = bodyValue.value;
                    }
                    break;
                }
            }
        }
        
        // Final fallback to preview text
        if (!bodyContent && email.preview) {
            bodyContent = email.preview;
        }
        
        if (bodyContent && bodyContent.trim()) {
            this.displayPreviewBody(bodyContent);
        } else {
            this.previewBody.innerHTML = '<div class="preview-loading">No content available</div>';
        }
    }

    displayPreviewBody(content) {
        // Clean and format the content
        let formattedContent = this.escapeHtml(content);
        
        // Show more content - truncate at 2000 characters instead of using full preview
        if (formattedContent.length > 2000) {
            formattedContent = formattedContent.substring(0, 2000) + '...';
        }
        
        // Convert line breaks to paragraphs with better formatting
        const paragraphs = formattedContent.split(/\n\s*\n/).filter(p => p.trim());
        
        if (paragraphs.length > 1) {
            formattedContent = paragraphs
                .map(p => `<p>${p.replace(/\n/g, '<br>')}</p>`)
                .join('');
        } else {
            formattedContent = `<p>${formattedContent.replace(/\n/g, '<br>')}</p>`;
        }
        
        // Apply some basic email content formatting
        formattedContent = this.enhanceEmailFormatting(formattedContent);
        
        this.previewBody.innerHTML = formattedContent;
    }

    positionPreviewPopup(event) {
        const popup = this.previewPopup;
        
        // Get mouse position relative to the page (including scroll offset)
        const mouseX = event.clientX + window.scrollX;
        const mouseY = event.clientY + window.scrollY;
        
        // Get viewport dimensions
        const viewportWidth = window.innerWidth;
        const viewportHeight = window.innerHeight;
        
        // Set initial position to measure popup dimensions
        popup.style.left = '0px';
        popup.style.top = '0px';
        
        // Get popup dimensions (it's now visible but off screen)
        const popupRect = popup.getBoundingClientRect();
        const popupWidth = Math.max(popupRect.width, 300); // minimum 300px width
        const popupHeight = Math.max(popupRect.height, 200); // minimum 200px height
        
        // Calculate position with offset from cursor
        const offset = 15;
        let left = mouseX + offset;
        let top = mouseY + offset;
        
        // Get current scroll position and viewport bounds
        const scrollX = window.scrollX;
        const scrollY = window.scrollY;
        const viewportRight = scrollX + viewportWidth;
        const viewportBottom = scrollY + viewportHeight;
        
        // Adjust if popup would go off screen horizontally
        if (left + popupWidth > viewportRight) {
            left = mouseX - popupWidth - offset;
        }
        
        // Adjust if popup would go off screen vertically
        if (top + popupHeight > viewportBottom) {
            top = mouseY - popupHeight - offset;
        }
        
        // Ensure popup doesn't go off screen entirely
        left = Math.max(scrollX + 10, Math.min(left, viewportRight - popupWidth - 10));
        top = Math.max(scrollY + 10, Math.min(top, viewportBottom - popupHeight - 10));
        
        // Apply the position
        popup.style.left = left + 'px';
        popup.style.top = top + 'px';
    }

    updatePreviewPosition(event) {
        if (this.previewPopup.style.display === 'block') {
            this.positionPreviewPopup(event);
        }
    }

    enhanceEmailFormatting(content) {
        // Add basic formatting for better readability
        let formatted = content;
        
        // Format URLs (make them look like links without actually being clickable)
        formatted = formatted.replace(/\b(https?:\/\/[\w\-\._~:\/?#\[\]@!\$&'\(\)*\+,;=.]+)/gi, 
            '<span style="color: #3498db; text-decoration: underline;">$1</span>');
        
        // Format email addresses  
        formatted = formatted.replace(/\b[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}\b/g, 
            '<span style="color: #3498db;">$&</span>');
        
        // Bold text patterns (common email signatures)
        formatted = formatted.replace(/\*\*([^*]+)\*\*/g, '<strong>$1</strong>');
        formatted = formatted.replace(/\*([^*]+)\*/g, '<em>$1</em>');
        
        // Format common patterns like "From:", "To:", "Subject:" in forwarded emails
        formatted = formatted.replace(/^(From|To|Subject|Date|CC|BCC):\s*/gim, 
            '<strong style="color: #666;">$1:</strong> ');
        
        return formatted;
    }
}

// Initialize the application when the DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new MailboxZero();
});