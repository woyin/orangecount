import { mount } from "svelte";

import App from "./App.svelte";
import "./styles/shell.css";

const target = document.getElementById("app");
if (!target) throw new Error("Fava shell mount target is missing");

mount(App, { target });
