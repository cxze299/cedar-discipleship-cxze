import { createApp } from 'vue';
import { createPinia } from 'pinia';
import App from './App.vue';
import { installVersionRefresh } from './runtime/appVersion';
import './styles.css';

installVersionRefresh();

const app = createApp(App);
app.use(createPinia());
app.mount('#app');
