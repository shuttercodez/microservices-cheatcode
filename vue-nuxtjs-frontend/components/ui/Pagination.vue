<script setup lang="ts">
defineProps<{
  page: number;
  totalPages: number;
}>();

defineEmits<{
  (e: "pageChange", newPage: number): void;
}>();
</script>

<template>
  <div
    class="px-6 py-4 bg-slate-50 border-t border-slate-200 flex justify-between items-center"
  >
    <button
      :disabled="page <= 1"
      class="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40 hover:bg-slate-50 transition"
      @click="$emit('pageChange', Math.max(1, page - 1))"
    >
      ← Previous
    </button>
    <span class="text-xs text-slate-600">
      Page {{ page }} of {{ totalPages || 1 }}
    </span>
    <button
      :disabled="page >= (totalPages || 1)"
      class="px-3 py-1.5 text-xs font-medium bg-white border border-slate-300 rounded-lg disabled:opacity-40 hover:bg-slate-50 transition"
      @click="$emit('pageChange', page + 1)"
    >
      Next →
    </button>
  </div>
</template>
