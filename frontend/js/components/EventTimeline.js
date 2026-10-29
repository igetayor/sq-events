/**
 * Component: EventTimeline displays the append-only chronological ledger.
 */
export class EventTimeline {
  constructor(containerId) {
    this.container = document.getElementById(containerId);
  }

  update(events = []) {
    if (!this.container) return;

    if (events.length === 0) {
      this.container.innerHTML = '<div style="color: var(--text-dim); text-align: center; padding: 2rem;">No events in ledger.</div>';
      return;
    }

    this.container.innerHTML = events.slice().reverse().map(evt => {
      let payloadStr = "";
      try {
        const p = typeof evt.payload === 'string' ? JSON.parse(evt.payload) : evt.payload;
        if (evt.event_type === "ACCOUNT_OPENED") {
          payloadStr = `Initial balance allocated: <strong>$${p.initial_balance}</strong> for ${p.owner}`;
        } else if (evt.event_type === "MONEY_DEPOSITED") {
          payloadStr = `Inflow: <strong style="color: #34d399;">+$${p.amount}</strong> &mdash; ${p.reason}`;
        } else if (evt.event_type === "MONEY_WITHDRAWN") {
          payloadStr = `Outflow: <strong style="color: #f97316;">-$${p.amount}</strong> &mdash; ${p.reason}`;
        } else if (evt.event_type === "ACCOUNT_FROZEN") {
          payloadStr = `Security lock applied by ${p.triggered_by || 'compliance'}`;
        } else if (evt.event_type === "ACCOUNT_UNFROZEN") {
          payloadStr = `Security lock released by compliance`;
        } else {
          payloadStr = JSON.stringify(p);
        }
      } catch (e) {
        payloadStr = JSON.stringify(evt.payload);
      }

      const dateStr = new Date(evt.timestamp * 1000).toLocaleTimeString();

      return `
        <div class="timeline-item ${evt.event_type}">
          <div class="timeline-top">
            <span class="event-type-badge">${evt.event_type}</span>
            <span style="color: var(--text-dim);">Seq #v${evt.version} &bull; ${dateStr}</span>
          </div>
          <div class="timeline-payload">
            ${payloadStr}
          </div>
        </div>
      `;
    }).join('');
  }
}
