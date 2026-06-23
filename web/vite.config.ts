import preact from "@preact/preset-vite";
import type { UserConfig } from "vite";

type VitePlusConfig = UserConfig & {
  staged?: Record<string, string>;
};

export default {
  plugins: [preact()],
  staged: {
    "*.{js,jsx,ts,tsx}": "vp check --fix",
    "*.{css,html,json,md,yaml,yml}": "vp fmt --write",
  },
} satisfies VitePlusConfig;
