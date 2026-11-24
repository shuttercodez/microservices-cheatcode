import { mount } from "@vue/test-utils";
import { describe, expect, it } from "vitest";
import LoginCard from "../components/auth/LoginCard.vue";

describe("LoginCard Vue component", () => {
  it("renders Register DB User tab when enableRegister is true", () => {
    const wrapper = mount(LoginCard, {
      props: {
        loading: false,
        enableRegister: true,
      },
    });

    expect(wrapper.text()).toContain("User Sign In");
    expect(wrapper.text()).toContain("Register DB User");
    expect(wrapper.text()).toContain("M2M Client");
  });

  it("hides Register DB User tab when enableRegister is false", () => {
    const wrapper = mount(LoginCard, {
      props: {
        loading: false,
        enableRegister: false,
      },
    });

    expect(wrapper.text()).toContain("User Sign In");
    expect(wrapper.text()).not.toContain("Register DB User");
    expect(wrapper.text()).toContain("M2M Client");
  });

  it("emits login event with password grant credentials on submit", async () => {
    const wrapper = mount(LoginCard, {
      props: {
        loading: false,
        enableRegister: true,
      },
    });

    await wrapper.find("form").trigger("submit.prevent");

    const emitted = wrapper.emitted("login");
    expect(emitted).toBeTruthy();
    expect(emitted?.[0]?.[0]).toEqual({
      grant_type: "password",
      username: "admin",
      password: "adminpassword",
    });
  });
});
