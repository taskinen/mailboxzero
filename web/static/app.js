class MailboxZero {
    constructor() {
        this.emails = [];
        this.similarEmails = [];
        this.selectedEmailId = null;
        this.selectedSimilarEmails = new Set();
        this.inboxSortBy = 'date'; // Default sort by date (newest first)
        this.similarSortBy = 'date'; // Default sort by date (newest first)
        this.totalInboxCount = 0; // Track total count from server
        
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
    }

    async loadEmails() {
        try {
            this.showLoading(this.inboxList, 'Loading emails...');
            
            const response = await fetch('/api/emails');
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            const inboxInfo = await response.json();
            this.emails = inboxInfo.emails;
            this.totalInboxCount = inboxInfo.totalCount;
            this.renderEmails(this.emails, this.inboxList, false);
            this.updateTitles();
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
        
        // Attach click listeners
        container.querySelectorAll('.email-item').forEach(item => {
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
}

// Initialize the application when the DOM is ready
document.addEventListener('DOMContentLoaded', () => {
    new MailboxZero();
});