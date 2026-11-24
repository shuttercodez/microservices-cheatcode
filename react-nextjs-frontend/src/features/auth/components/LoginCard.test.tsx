import React from "react";
import { fireEvent, render, screen } from "@testing-library/react";
import { describe, expect, it, vi } from "vitest";
import { LoginCard } from "./LoginCard";

describe("LoginCard component", () => {
  it("renders Register DB User tab when enableRegister is true", () => {
    render(
      <LoginCard
        onLogin={vi.fn()}
        onRegister={vi.fn()}
        enableRegister={true}
        loading={false}
      />
    );

    expect(screen.getByText("User Sign In")).toBeInTheDocument();
    expect(screen.getByText("Register DB User")).toBeInTheDocument();
    expect(screen.getByText("M2M Client")).toBeInTheDocument();
  });

  it("hides Register DB User tab when enableRegister is false", () => {
    render(
      <LoginCard
        onLogin={vi.fn()}
        onRegister={vi.fn()}
        enableRegister={false}
        loading={false}
      />
    );

    expect(screen.getByText("User Sign In")).toBeInTheDocument();
    expect(screen.queryByText("Register DB User")).not.toBeInTheDocument();
    expect(screen.getByText("M2M Client")).toBeInTheDocument();
  });

  it("submits password grant credentials via onLogin", () => {
    const onLogin = vi.fn().mockResolvedValue(undefined);
    render(
      <LoginCard
        onLogin={onLogin}
        onRegister={vi.fn()}
        enableRegister={true}
        loading={false}
      />
    );

    fireEvent.click(screen.getByText("Get OAuth 2.0 Access Token"));

    expect(onLogin).toHaveBeenCalledWith({
      grant_type: "password",
      username: "admin",
      password: "adminpassword",
    });
  });
});
