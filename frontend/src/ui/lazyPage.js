import { defineAsyncComponent } from 'vue';
import AsyncPageStatus from './AsyncPageStatus.vue';

export function lazyPage(loader) {
  return defineAsyncComponent({
    loader,
    loadingComponent: AsyncPageStatus,
    errorComponent: AsyncPageStatus,
  });
}
