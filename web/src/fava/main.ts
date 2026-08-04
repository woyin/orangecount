import { mount } from "svelte";

import App from "./App.svelte";
// Fava 1.30.12 global styles, adapted only at the OrangeCount component seam.
import "./styles/fava/style.css";
import "./styles/fava/base.css";
import "./styles/fava/layout.css";
import "./styles/fava/components.css";
import "./styles/fava/fonts.css";

const target = document.getElementById("app");
if (!target) throw new Error("Fava shell mount target is missing");

mount(App, { target });
