<script setup lang="ts">
import type { Order } from "~/types/order";

defineProps<{
  orders: Order[];
  page: number;
  totalPages: number;
  total: number;
}>();

defineEmits<{
  (e: "pageChange", page: number): void;
}>();
</script>

<template>
  <div
    class="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden"
  >
    <div
      class="px-6 py-4 border-b border-slate-200 flex justify-between items-center"
    >
      <h2 class="text-lg font-semibold text-slate-900">
        Customer Orders (Redis Cached)
      </h2>
      <span class="text-xs text-slate-500">
        Page {{ page }} of {{ totalPages || 1 }} ({{ total }} orders)
      </span>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead
          class="bg-slate-50 text-xs uppercase text-slate-500 border-b border-slate-200"
        >
          <tr>
            <th class="px-6 py-3">Order ID</th>
            <th class="px-4 py-3">Product ID</th>
            <th class="px-4 py-3">Qty</th>
            <th class="px-4 py-3">Total Price</th>
            <th class="px-6 py-3">Status</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-200">
          <tr v-if="orders.length === 0">
            <td colspan="5" class="px-6 py-8 text-center text-slate-400">
              No orders placed yet.
            </td>
          </tr>
          <tr v-for="o in orders" :key="o.id" class="hover:bg-slate-50">
            <td class="px-6 py-3 font-mono text-xs text-slate-700">
              {{ o.id }}
            </td>
            <td class="px-4 py-3 font-mono text-xs text-slate-500">
              {{ o.product_id }}
            </td>
            <td class="px-4 py-3 font-semibold">{{ o.quantity }}</td>
            <td class="px-4 py-3 font-semibold text-emerald-700">
              ${{ o.total_price.toLocaleString() }}
            </td>
            <td class="px-6 py-3">
              <span
                class="px-2.5 py-0.5 rounded-full text-xs font-semibold bg-emerald-100 text-emerald-800"
              >
                {{ o.status }}
              </span>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <UiPagination
      :page="page"
      :total-pages="totalPages"
      @page-change="$emit('pageChange', $event)"
    />
  </div>
</template>
