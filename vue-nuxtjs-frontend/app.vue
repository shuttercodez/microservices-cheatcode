<script setup lang="ts">
import { onMounted, onUnmounted, watch } from "vue";
import { AUTH_EXPIRED_EVENT } from "~/utils/api-client";

const router = useRouter();
const {
  apiBaseUrl,
  health,
  token,
  isAuthenticated,
  authLoading,
  errorMsg,
  successMsg,
  showError,
  showSuccess,
  refreshHealth,
  fetchAuthConfig,
  syncTokenFromStorage,
  logout,
} = useConsole();

function handleExpiredEvent(event: Event) {
  const customEvent = event as CustomEvent<string>;
  token.value = null;
  showError(
    customEvent.detail || "Your session has expired. Please sign in again."
  );
  router.replace("/login");
}

onMounted(() => {
  syncTokenFromStorage();
  refreshHealth();
  fetchAuthConfig();
  if (typeof window !== "undefined") {
    window.addEventListener(AUTH_EXPIRED_EVENT, handleExpiredEvent);
  }
});

onUnmounted(() => {
  if (typeof window !== "undefined") {
    window.removeEventListener(AUTH_EXPIRED_EVENT, handleExpiredEvent);
  }
});

watch(apiBaseUrl, () => {
  refreshHealth();
  fetchAuthConfig();
});

function handleLogout() {
  logout();
  showSuccess("Logged out and cleared JWT token");
  router.replace("/login");
}
</script>

<template>
  <div class="min-h-screen bg-slate-50 text-slate-900 antialiased">
    <div class="max-w-6xl mx-auto px-4 py-8">
      <LayoutNavbar
        v-model:api-base-url="apiBaseUrl"
        :health="health"
        :loading="authLoading"
        :is-authenticated="isAuthenticated"
        @refresh="refreshHealth"
        @logout="handleLogout"
      />

      <UiAlertBanner
        type="error"
        :message="errorMsg"
        @dismiss="errorMsg = null"
      />
      <UiAlertBanner
        type="success"
        :message="successMsg"
        @dismiss="successMsg = null"
      />

      <NuxtPage />
    </div>
  </div>
</template>
