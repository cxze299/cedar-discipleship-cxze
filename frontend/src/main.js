import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import { installVersionRefresh } from './runtime/appVersion';
import './styles.css';
import './styles/fonts.css';
import './styles/tokens.css';
import './styles/base.css';
import './styles/layout.css';
import './styles/composition.css';

installVersionRefresh();

const app = createApp(App);
const pinia = createPinia();
app.use(pinia);
app.mount('#app');

if (typeof window !== 'undefined') {
  window.__pinia__ = pinia;
}
