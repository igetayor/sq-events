import { Api } from './api.js';
import { AccountCard } from './components/AccountCard.js';
import { EventTimeline } from './components/EventTimeline.js';
import { CommandPanel } from './components/CommandPanel.js';

class EventSourcingApp {
  constructor() {
    this.accountCard = new AccountCard('accountHeroContainer');
    this.timeline = new EventTimeline('timelineStream');
    this.currentVersion = 0;

    this.commandPanel = new CommandPanel('commandForm', (cmd) => this.handleCommand(cmd));
  }

  async init() {
    this.bindEvents();
    await this.refresh();
  }

  bindEvents() {
    document.addEventListener('click', async (e) => {
      if (e.target && e.target.id === 'btnResetStore') {
        await Api.reset();
        await this.refresh();
      }
    });
  }

  async handleCommand(cmd) {
    const errorBanner = document.getElementById('errorBanner');
    errorBanner.style.display = 'none';

    try {
      const res = await Api.appendEvent({
        stream_id: "acc_prime_001",
        event_type: cmd.event_type,
        amount: cmd.amount,
        reason: cmd.reason,
        expected_version: this.currentVersion
      });

      this.currentVersion = res.state.version;
      this.accountCard.update(res.state);
      await this.refresh();
    } catch (err) {
      errorBanner.textContent = `❌ ${err.message}`;
      errorBanner.style.display = 'block';
    }
  }

  async refresh() {
    const data = await Api.getAccount("acc_prime_001");
    if (!data) return;

    this.currentVersion = data.state.version;
    this.accountCard.update(data.state);
    this.timeline.update(data.events);
  }
}

document.addEventListener('DOMContentLoaded', () => {
  const app = new EventSourcingApp();
  app.init();
});
