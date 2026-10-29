/**
 * Component: AccountCard displays the folded aggregate state (Balance, Version, Status).
 */
export class AccountCard {
  constructor(containerId) {
    this.container = document.getElementById(containerId);
  }

  update(state) {
    if (!this.container || !state) return;

    const formattedBalance = new Intl.NumberFormat('en-US', {
      style: 'currency',
      currency: 'USD'
    }).format(state.balance || 0);

    this.container.innerHTML = `
      <div class="account-balance-group">
        <div class="balance-title">Rehydrated Projected Balance</div>
        <div class="balance-amount ${state.status}">
          ${formattedBalance}
        </div>
        <div style="font-size: 0.85rem; color: var(--text-muted);">
          Entity: <strong style="color: #fff;">${state.owner || 'Vault'}</strong> 
          <span style="color: var(--text-dim); margin-left: 0.5rem;">(${state.account_id})</span>
        </div>
      </div>

      <div style="display: flex; flex-direction: column; align-items: flex-end; gap: 0.75rem;">
        <span class="status-badge ${state.status}">
          STATUS: ${state.status}
        </span>
        <div style="font-family: var(--font-mono); font-size: 0.85rem; color: var(--text-dim);">
          Aggregate Version: <strong style="color: #38bdf8;">v${state.version || 0}</strong>
        </div>
        <button id="btnResetStore" style="background: none; border: 1px solid var(--border-subtle); color: var(--text-dim); font-size: 0.75rem; padding: 0.35rem 0.65rem; border-radius: 4px; cursor: pointer;">
          ↺ Clear Ledger
        </button>
      </div>
    `;
  }
}
