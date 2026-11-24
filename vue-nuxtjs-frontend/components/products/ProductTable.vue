<script setup lang="ts">
import { ref } from "vue";
import type { Product } from "~/types/product";

defineProps<{
  products: Product[];
  page: number;
  totalPages: number;
  total: number;
}>();

const emit = defineEmits<{
  (e: "pageChange", page: number): void;
  (e: "patchStock", id: string, newStock: number): void;
  (e: "delete", product: Product): void;
  (e: "selectForOrder", productId: string): void;
}>();

const editingStockId = ref<string | null>(null);
const newStockValue = ref<number | string>("");

function handleSaveStock(productId: string) {
  const parsed = Number(newStockValue.value);
  if (Number.isNaN(parsed)) return;
  emit("patchStock", productId, parsed);
  editingStockId.value = null;
}
</script>

<template>
  <div
    class="lg:col-span-2 bg-white rounded-xl shadow-sm border border-slate-200 overflow-hidden"
  >
    <div
      class="px-6 py-4 border-b border-slate-200 flex justify-between items-center"
    >
      <h2 class="text-lg font-semibold text-slate-900">
        Product Inventory (Redis Cached)
      </h2>
      <span class="text-xs text-slate-500">
        Page {{ page }} of {{ totalPages || 1 }} ({{ total }} items)
      </span>
    </div>

    <div class="overflow-x-auto">
      <table class="w-full text-left text-sm">
        <thead
          class="bg-slate-50 text-xs uppercase text-slate-500 border-b border-slate-200"
        >
          <tr>
            <th class="px-6 py-3">SKU / Name</th>
            <th class="px-4 py-3">Price</th>
            <th class="px-4 py-3">Stock</th>
            <th class="px-4 py-3">Version</th>
            <th class="px-6 py-3 text-right">Actions</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-slate-200">
          <tr v-if="products.length === 0">
            <td colspan="5" class="px-6 py-8 text-center text-slate-400">
              No products found. Create your first product on the left.
            </td>
          </tr>
          <tr v-for="p in products" :key="p.id" class="hover:bg-slate-50">
            <td class="px-6 py-3">
              <div class="font-semibold text-slate-900">{{ p.sku }}</div>
              <div class="text-xs text-slate-500">{{ p.name }}</div>
              <div class="text-[11px] text-slate-400 font-mono">
                {{ p.id }}
              </div>
            </td>
            <td class="px-4 py-3 font-medium">
              ${{ p.price.toLocaleString() }}
            </td>
            <td class="px-4 py-3">
              <div
                v-if="editingStockId === p.id"
                class="flex items-center gap-1"
              >
                <input
                  v-model="newStockValue"
                  type="number"
                  min="0"
                  class="w-20 border border-slate-300 rounded px-2 py-1 text-xs"
                />
                <button
                  class="bg-emerald-600 text-white text-xs px-2 py-1 rounded"
                  @click="handleSaveStock(p.id)"
                >
                  Save
                </button>
                <button
                  class="text-slate-400 text-xs px-1"
                  @click="editingStockId = null"
                >
                  ✕
                </button>
              </div>
              <span
                v-else
                :class="[
                  'inline-flex items-center px-2.5 py-0.5 rounded-full text-xs font-medium',
                  p.stock > 5
                    ? 'bg-emerald-100 text-emerald-800'
                    : p.stock > 0
                    ? 'bg-amber-100 text-amber-800'
                    : 'bg-rose-100 text-rose-800',
                ]"
              >
                {{ p.stock }} in stock
              </span>
            </td>
            <td class="px-4 py-3 text-xs text-slate-500 font-mono">
              v{{ p.version }}
            </td>
            <td class="px-6 py-3 text-right space-x-2">
              <button
                class="text-xs text-blue-600 hover:text-blue-800 font-medium"
                @click="
                  editingStockId = p.id;
                  newStockValue = p.stock;
                "
              >
                Patch Stock
              </button>
              <button
                class="text-xs text-emerald-600 hover:text-emerald-800 font-medium"
                @click="$emit('selectForOrder', p.id)"
              >
                Order
              </button>
              <button
                class="text-xs text-rose-600 hover:text-rose-800 font-medium"
                @click="$emit('delete', p)"
              >
                Delete
              </button>
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
