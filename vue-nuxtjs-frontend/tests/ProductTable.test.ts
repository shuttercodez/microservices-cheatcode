import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import ProductTable from "../components/products/ProductTable.vue";
import Pagination from "../components/ui/Pagination.vue";
import type { Product } from "../types/product";

const sampleProducts: Product[] = [
  {
    id: "550e8400-e29b-41d4-a716-446655440000",
    sku: "IPHONE-16",
    name: "iPhone 16 Pro",
    price: 39900,
    stock: 25,
    version: 3,
    created_at: "2026-11-24T00:00:00Z",
    updated_at: "2026-11-24T00:00:00Z",
  },
];

describe("ProductTable Vue component", () => {
  it("renders product rows with SKU, stock badge, and optimistic lock version", () => {
    const wrapper = mount(ProductTable, {
      props: {
        products: sampleProducts,
        page: 1,
        totalPages: 2,
        total: 15,
      },
      global: {
        components: {
          UiPagination: Pagination,
        },
      },
    });

    expect(wrapper.text()).toContain("IPHONE-16");
    expect(wrapper.text()).toContain("iPhone 16 Pro");
    expect(wrapper.text()).toContain("25 in stock");
    expect(wrapper.text()).toContain("v3");
  });

  it("emits selectForOrder event when Order button is clicked", async () => {
    const wrapper = mount(ProductTable, {
      props: {
        products: sampleProducts,
        page: 1,
        totalPages: 1,
        total: 1,
      },
      global: {
        components: {
          UiPagination: Pagination,
        },
      },
    });

    const buttons = wrapper.findAll("button");
    const orderBtn = buttons.find((b) => b.text() === "Order");
    await orderBtn?.trigger("click");

    const emitted = wrapper.emitted("selectForOrder");
    expect(emitted).toBeTruthy();
    expect(emitted?.[0]?.[0]).toBe(sampleProducts[0].id);
  });
});
