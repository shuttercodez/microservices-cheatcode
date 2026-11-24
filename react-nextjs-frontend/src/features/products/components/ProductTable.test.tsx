import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { ProductTable } from "./ProductTable";
import { Product } from "../types/product";

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

describe("ProductTable component", () => {
  it("renders product rows with SKU, stock badge, and optimistic lock version", () => {
    render(
      <ProductTable
        products={sampleProducts}
        page={1}
        totalPages={2}
        total={15}
        onPageChange={vi.fn()}
        onPatchStock={vi.fn()}
        onDelete={vi.fn()}
        onSelectForOrder={vi.fn()}
      />
    );

    expect(screen.getByText("IPHONE-16")).toBeInTheDocument();
    expect(screen.getByText("iPhone 16 Pro")).toBeInTheDocument();
    expect(screen.getByText("25 in stock")).toBeInTheDocument();
    expect(screen.getByText("v3")).toBeInTheDocument();
  });

  it("navigates to order route when Order action is clicked", () => {
    const onSelectForOrder = vi.fn();
    render(
      <ProductTable
        products={sampleProducts}
        page={1}
        totalPages={1}
        total={1}
        onPageChange={vi.fn()}
        onPatchStock={vi.fn()}
        onDelete={vi.fn()}
        onSelectForOrder={onSelectForOrder}
      />
    );

    fireEvent.click(screen.getByText("Order"));
    expect(onSelectForOrder).toHaveBeenCalledWith(sampleProducts[0].id);
  });
});
