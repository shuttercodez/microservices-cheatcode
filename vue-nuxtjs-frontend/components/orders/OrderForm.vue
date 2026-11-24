<script setup lang="ts">
import { ref } from "vue";
import type { CreateOrderRequest } from "~/types/order";
import type { Product } from "~/types/product";

const props = defineProps<{
  products: Product[];
  selectedProductId: string;
}>();

const emit = defineEmits<{
  (e: "update:selectedProductId", id: string): void;
  (e: "create", req: CreateOrderRequest): void;
}>();

const quantity = ref<number | string>(1);

function handleSubmit() {
  emit("create", {
    product_id: props.selectedProductId.trim(),
    quantity: Number(quantity.value) || 1,
  });
  quantity.value = 1;
}
</script>

<template>
  <div
    class="bg-white rounded-xl shadow-sm border border-slate-200 p-6 h-fit"
  >
    <h2 class="text-lg font-semibold text-slate-900 mb-4">
      Place Customer Order
    </h2>
    <form class="space-y-4" @submit.prevent="handleSubmit">
      <div>
        <label class="block text-xs font-medium text-slate-700 mb-1">
          Select Product (or enter UUID)
        </label>
        <select
          :value="selectedProductId"
          class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2 mb-2 bg-white"
          @change="
            $emit(
              'update:selectedProductId',
              ($event.target as HTMLSelectElement).value
            )
          "
        >
          <option value="">-- Choose from current page --</option>
          <option v-for="p in products" :key="p.id" :value="p.id">
            {{ p.sku }} — {{ p.name }} (${{ p.price }}, Stock: {{ p.stock }})
          </option>
        </select>
        <input
          :value="selectedProductId"
          type="text"
          required
          placeholder="Product UUID"
          class="w-full text-xs font-mono border border-slate-300 rounded-lg px-3 py-2"
          @input="
            $emit(
              'update:selectedProductId',
              ($event.target as HTMLInputElement).value
            )
          "
        />
      </div>
      <div>
        <label class="block text-xs font-medium text-slate-700 mb-1">
          Quantity
        </label>
        <input
          v-model="quantity"
          type="number"
          min="1"
          required
          class="w-full text-sm border border-slate-300 rounded-lg px-3 py-2"
        />
      </div>
      <button
        type="submit"
        class="w-full bg-emerald-600 hover:bg-emerald-700 text-white text-sm font-medium py-2.5 rounded-lg transition"
      >
        Place Order &amp; Deduct Stock
      </button>
    </form>
  </div>
</template>
