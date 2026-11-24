<script setup lang="ts">
import type { HealthStatus } from "~/types/common";

defineProps<{
  apiBaseUrl: string;
  health: HealthStatus;
  loading: boolean;
  isAuthenticated: boolean;
}>();

defineEmits<{
  (e: "update:apiBaseUrl", url: string): void;
  (e: "refresh"): void;
  (e: "logout"): void;
}>();
</script>

<template>
  <header
    class="bg-white rounded-xl shadow-sm border border-slate-200 p-6 mb-6 flex flex-col md:flex-row md:items-center md:justify-between gap-4"
  >
    <div>
      <div class="flex flex-wrap items-center gap-2">
        <span
          class="px-2.5 py-0.5 text-xs font-semibold bg-emerald-100 text-emerald-800 rounded-full"
        >
          Vue 3 + Nuxt 3
        </span>
        <span
          :class="[
            'px-2.5 py-0.5 text-xs font-semibold rounded-full',
            health.alive
              ? 'bg-emerald-100 text-emerald-800'
              : 'bg-rose-100 text-rose-800',
          ]"
        >
          /healthz: {{ health.alive ? "ALIVE" : "OFFLINE" }}
        </span>
        <span
          :class="[
            'px-2.5 py-0.5 text-xs font-semibold rounded-full',
            health.ready
              ? 'bg-emerald-100 text-emerald-800'
              : 'bg-amber-100 text-amber-800',
          ]"
        >
          /readyz: {{ health.ready ? "READY" : "NOT READY" }}
        </span>
        <span
          :class="[
            'px-2.5 py-0.5 text-xs font-semibold rounded-full',
            isAuthenticated
              ? 'bg-indigo-100 text-indigo-800'
              : 'bg-slate-100 text-slate-600',
          ]"
        >
          {{ isAuthenticated ? "🔐 JWT Active" : "🔒 Unauthenticated" }}
        </span>
      </div>
      <h1 class="text-2xl font-bold text-slate-900 mt-2">
        Inventory &amp; Order Microservice Console
      </h1>
      <p class="text-sm text-slate-500">
        Secured with OAuth 2.0 + HS256 JWT across Go, Rust, and Python
      </p>
    </div>

    <div class="flex items-center gap-2">
      <input
        :value="apiBaseUrl"
        type="text"
        class="text-sm border border-slate-300 rounded-lg px-3 py-2 w-56 focus:outline-none focus:ring-2 focus:ring-emerald-500"
        placeholder="http://localhost:8080"
        @input="
          $emit(
            'update:apiBaseUrl',
            ($event.target as HTMLInputElement).value
          )
        "
      />
      <button
        :disabled="loading"
        class="bg-slate-900 hover:bg-slate-800 text-white text-sm font-medium px-4 py-2 rounded-lg transition"
        @click="$emit('refresh')"
      >
        {{ loading ? "Syncing..." : "Refresh" }}
      </button>
      <button
        v-if="isAuthenticated"
        class="bg-rose-50 hover:bg-rose-100 text-rose-700 border border-rose-200 text-sm font-medium px-3 py-2 rounded-lg transition"
        @click="$emit('logout')"
      >
        Logout
      </button>
    </div>
  </header>
</template>
