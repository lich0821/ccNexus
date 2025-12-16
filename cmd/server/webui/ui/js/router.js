// Simple client-side router
import { state } from './state.js';

class Router {
    constructor() {
        this.routes = new Map();
        this.currentView = null;
    }

    register(name, component) {
        this.routes.set(name, component);
    }

    navigate(viewName) {
        if (!this.routes.has(viewName)) {
            console.error(`View "${viewName}" not found`);
            return;
        }

        // Update active nav link
        document.querySelectorAll('.nav-link').forEach(link => {
            link.classList.remove('active');
            if (link.dataset.view === viewName) {
                link.classList.add('active');
            }
        });

        // Update state
        state.update('currentView', viewName);

        // Render view
        const component = this.routes.get(viewName);
        this.currentView = component;
        component.render();
    }

    init() {
        // Set up nav link click handlers
        document.querySelectorAll('.nav-link').forEach(link => {
            link.addEventListener('click', (e) => {
                e.preventDefault();
                const viewName = link.dataset.view;
                this.navigate(viewName);
            });
        });

        // Navigate to initial view
        const initialView = state.get('currentView');
        this.navigate(initialView);
    }
}

export const router = new Router();
