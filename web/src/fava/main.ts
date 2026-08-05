import { mount } from "svelte";

import App from "./App.svelte";
// Fava 1.30.12 global styles, adapted only at the OrangeCount component seam.
import "./styles/fava/style.css";
import "./styles/fava/base.css";
import "./styles/fava/layout.css";
import "./styles/fava/components.css";
import "./styles/fava/fonts.css";
import "./styles/fava/grid.css";
import "./styles/fava/journal-table.css";
import "./styles/fava/tree-table.css";
import "./styles/fava/charts.css";
import "./styles/fava/editor.css";
import "./styles/fava/help.css";
import "./styles/fava/notifications.css";
// Loaded last so OrangeCount-owned overrides win over the derived stylesheets.
import "./styles/orangecount.css";

const target = document.getElementById("app");
if (!target) throw new Error("Fava shell mount target is missing");

mount(App, { target });
