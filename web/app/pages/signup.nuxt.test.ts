import { mountSuspended } from "@nuxt/test-utils/runtime"
import { beforeEach, describe, it } from "vitest"
import { clearQueryCache } from "~/composables/api/test-helpers"
import Signup from "./signup.vue"
import { findButtonByText, seedAuthConfig, t } from "../components/test-helpers"

// the query cache is an app-wide singleton every mount in the file shares
describe("<signup>", { concurrent: false }, () => {
	beforeEach(clearQueryCache)

	it("shows the email/password form without a way back when it is the only enabled method", async ({
		expect,
	}) => {
		seedAuthConfig({
			methods: ["email-password"],
		})

		const wrapper = await mountSuspended(Signup)

		expect(wrapper.find('input[type="email"]').exists()).toBe(true)
		expect(wrapper.text()).not.toContain(
			t("onboarding.signup.email-password-form.back"),
		)
	})

	it("goes back to the methods from the email/password form", async ({
		expect,
	}) => {
		seedAuthConfig({
			methods: ["google", "email-password"],
		})
		const wrapper = await mountSuspended(Signup)
		await findButtonByText(
			wrapper,
			t("onboarding.signup.signup-email-password"),
		).trigger("click")

		await findButtonByText(
			wrapper,
			t("onboarding.signup.email-password-form.back"),
		).trigger("click")

		expect(wrapper.find('input[type="email"]').exists()).toBe(false)
		expect(wrapper.text()).toContain(t("onboarding.signup.signup-google"))
	})
})
