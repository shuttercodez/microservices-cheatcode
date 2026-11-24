<script setup lang="ts">
defineProps<{
  type: "error" | "success";
  message: string | null;
}>();

defineEmits<{
  (e: "dismiss"): void;
}>();
</script>

<template>
  <div
    v-if="message"
    :class="[
      'mb-6 px-4 py-3 rounded-lg text-sm flex justify-between items-center border',
      type === 'error'
        ? 'bg-rose-50 border-rose-200 text-rose-700'
        : 'bg-emerald-50 border-emerald-200 text-emerald-800',
    ]"
  >
    <span>{{ type === "error" ? "⚠️" : "✅" }} {{ message }}</span>
    <button
      :class="[
        'font-bold',
        type === 'error'
          ? 'text-rose-500 hover:text-rose-800'
          : 'text-emerald-600 hover:text-emerald-900',
      ]"
      @click="$emit('dismiss')"
    >
      ✕
    </button>
  </div>
</template>
