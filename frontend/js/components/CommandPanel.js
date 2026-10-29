/**
 * Component: CommandPanel provides interactive command triggers (Deposit, Withdraw, Freeze, Unfreeze).
 */
export class CommandPanel {
  constructor(formId, onCommandCallback) {
    this.form = document.getElementById(formId);
    this.onCommand = onCommandCallback;
    this.init();
  }

  init() {
    document.getElementById('btnDeposit')?.addEventListener('click', () => {
      this.triggerCommand('MONEY_DEPOSITED');
    });

    document.getElementById('btnWithdraw')?.addEventListener('click', () => {
      this.triggerCommand('MONEY_WITHDRAWN');
    });

    document.getElementById('btnFreeze')?.addEventListener('click', () => {
      this.triggerCommand('ACCOUNT_FROZEN');
    });

    document.getElementById('btnUnfreeze')?.addEventListener('click', () => {
      this.triggerCommand('ACCOUNT_UNFROZEN');
    });
  }

  triggerCommand(eventType) {
    const amount = parseFloat(document.getElementById('txnAmount').value) || 0;
    const reason = document.getElementById('txnReason').value || "Manual Transaction";

    if ((eventType === 'MONEY_DEPOSITED' || eventType === 'MONEY_WITHDRAWN') && amount <= 0) {
      alert("Please enter a positive transaction amount.");
      return;
    }

    if (this.onCommand) {
      this.onCommand({
        event_type: eventType,
        amount: amount,
        reason: reason
      });
    }
  }
}
