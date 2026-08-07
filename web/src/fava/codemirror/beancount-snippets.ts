import type { Completion } from "@codemirror/autocomplete";
import { snippetCompletion } from "@codemirror/autocomplete";

const todayAsString = () => new Date().toISOString().slice(0, 10);

export const beancount_snippets: () => readonly Completion[] = () => {
  const today = todayAsString();
  return [
    snippetCompletion(
      `${today} #{*} "#{}" "#{}"\n  #{Account:A}  #{Amount}\n  #{Account:B}`,
      {
        label: `${today} * transaction`,
      },
    ),
  ];
};
