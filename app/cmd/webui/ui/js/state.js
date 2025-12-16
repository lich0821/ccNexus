// Simple reactive state management
class State {
    constructor() {
        this.data = {
            endpoints: [],
            stats: null,
            config: null,
            currentView: 'dashboard',
            currentEndpoint: null
        };
        this.listeners = new Map();
    }

    subscribe(key, callback) {
        if (!this.listeners.has(key)) {
            this.listeners.set(key, []);
        }
        this.listeners.get(key).push(callback);

        // Return unsubscribe function
        return () => {
            const callbacks = this.listeners.get(key);
            const index = callbacks.indexOf(callback);
            if (index > -1) {
                callbacks.splice(index, 1);
            }
        };
    }

    update(key, value) {
        this.data[key] = value;
        if (this.listeners.has(key)) {
            this.listeners.get(key).forEach(cb => cb(value));
        }
    }

    get(key) {
        return this.data[key];
    }
}

export const state = new State();
