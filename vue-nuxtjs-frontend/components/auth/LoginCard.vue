<script setup lang="ts">
import { ref } from "vue";
import type { OAuthTokenRequest, RegisterUserRequest } from "~/types/auth";

type AuthMode = "password" | "register" | "client_credentials";

defineProps<{
  loading: boolean;
  enableRegister: boolean;
}>();

const emit = defineEmits<{
  (e: "login", req: OAuthTokenRequest): void;
  (e: "register", req: RegisterUserRequest): void;
}>();

const mode = ref<AuthMode>("password");
const username = ref("admin");
const password = ref("adminpassword");
const role = ref("user");
const clientId = ref("inventory-client");
const clientSecret = ref("inventory-secret");

function switchToRegister() {
  mode.value = "register";
  if (username.value === "admin") {
    username.value = "";
    password.value = "";
  }
}

function handleSubmit() {
  if (mode.value === "password") {
    emit("login", {
      grant_type: "password",
      username: username.value.trim(),
      password: password.value,
    });
  } else if (mode.value === "register") {
    emit("register", {
      username: username.value.trim(),
      password: password.value,
      role: role.value,
    });
  } else {
    emit("login", {
      grant_type: "client_credentials",
      client_id: clientId.value.trim(),
      client_secret: clientSecret.value,
    });
  }
}
</script>

<template>
  <div
    class="max-w-md mx-auto bg-white rounded-xl shadow-sm border border-slate-200 p-6 my-8"
  >
    <div class="flex items-center justify-between mb-4">
      <h2 class="text-lg font-bold text-slate-900">
        🔐 OAuth 2.0 + PostgreSQL Auth
      </h2>
      <span
        class="text-xs px-2 py-0.5 bg-slate-100 text-slate-600 rounded font-mono"
      >
        {{ mode === "register" ? "POST /oauth/register" : "POST /oauth/token" }}
      </span>
    </div>

    <!-- Mode Switcher -->
    <div
      :class="[
        'grid gap-1.5 p-1 bg-slate-100 rounded-lg mb-4 text-xs font-medium',
        enableRegister ? 'grid-cols-3' : 'grid-cols-2',
      ]"
    >
      <button
        type="button"
        :class="[
          'py-1.5 rounded-md transition',
          mode === 'password'
            ? 'bg-white text-slate-900 shadow-sm'
            : 'text-slate-600',
        ]"
        @click="mode = 'password'"
      >
        User Sign In
      </button>
      <button
        v-if="enableRegister"
        type="button"
        :class="[
          'py-1.5 rounded-md transition',
          mode === 'register'
            ? 'bg-white text-slate-900 shadow-sm'
            : 'text-slate-600',
        ]"
        @click="switchToRegister"
      >
        Register DB User
      </button>
      <button
        type="button"
        :class="[
          'py-1.5 rounded-md transition',
          mode === 'client_credentials'
            ? 'bg-white text-slate-900 shadow-sm'
            : 'text-slate-600',
        ]"
        @click="mode = 'client_credentials'"
      >
        M2M Client
      </button>
    </div>

    <form class="space-y-4" @submit.prevent="handleSubmit">
      <template v-if="mode === 'password' || mode === 'register'">
        <div>
          <label class="block text-xs font-medium text-slate-700 mb-1">
            Username (stored in PostgreSQL `users` table)
          </label>
          <input
            v-model="username"
            type="text"
            required
            placeholder="Enter username"
            class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
          />
        </div>
        <div>
          <label class="block text-xs font-medium text-slate-700 mb-1">
            Password (hashed with Bcrypt in DB)
          </label>
          <input
            v-model="password"
            type="password"
            required
            minlength="6"
            placeholder="Min 6 characters"
            class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
          />
        </div>
        <div v-if="mode === 'register'">
          <label class="block text-xs font-medium text-slate-700 mb-1">
            Role
          </label>
          <select
            v-model="role"
            class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 bg-white"
          >
            <option value="user">user</option>
            <option value="admin">admin</option>
          </select>
        </div>
      </template>

      <template v-else>
        <div>
          <label class="block text-xs font-medium text-slate-700 mb-1">
            Client ID (stored in PostgreSQL `oauth_clients` table)
          </label>
          <input
            v-model="clientId"
            type="text"
            required
            class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 font-mono"
          />
        </div>
        <div>
          <label class="block text-xs font-medium text-slate-700 mb-1">
            Client Secret (hashed with Bcrypt in DB)
          </label>
          <input
            v-model="clientSecret"
            type="password"
            required
            class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 font-mono"
          />
        </div>
      </template>

      <button
        type="submit"
        :disabled="loading"
        class="w-full bg-emerald-600 hover:bg-emerald-700 disabled:opacity-50 text-white text-sm font-semibold py-2.5 rounded-lg transition"
      >
        {{
          loading
            ? "Processing..."
            : mode === "register"
            ? "Create Account & Sign In"
            : "Get OAuth 2.0 Access Token"
        }}
      </button>
    </form>
  </div>
</template>
