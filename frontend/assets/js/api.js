/**
 * API client module for Event Sourcing ledger.
 */
export const Api = {
  async getAccount(streamId = "acc_prime_001") {
    try {
      const res = await fetch(`/api/account?stream_id=${encodeURIComponent(streamId)}`);
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      return await res.json();
    } catch (err) {
      console.warn("Failed to fetch account ledger:", err);
      return null;
    }
  },

  async appendEvent(command) {
    const res = await fetch('/api/events', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(command)
    });
    const data = await res.json();
    if (!res.ok) {
      throw new Error(data.error || `HTTP ${res.status}`);
    }
    return data;
  },

  async reset() {
    const res = await fetch('/api/reset', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' }
    });
    return await res.json();
  }
};
