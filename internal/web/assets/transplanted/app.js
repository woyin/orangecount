// node_modules/esm-env/dev-fallback.js
var node_env = globalThis.process?.env?.NODE_ENV;
var dev_fallback_default = node_env && !node_env.toLowerCase().startsWith("prod");

// node_modules/svelte/src/internal/shared/utils.js
var is_array = Array.isArray;
var array_from = Array.from;
var object_keys = Object.keys;
var define_property = Object.defineProperty;
var get_descriptor = Object.getOwnPropertyDescriptor;
var get_descriptors = Object.getOwnPropertyDescriptors;
var object_prototype = Object.prototype;
var array_prototype = Array.prototype;
var get_prototype_of = Object.getPrototypeOf;
var noop = () => {
};
function run(fn) {
  return fn();
}
function run_all(arr) {
  for (var i = 0; i < arr.length; i++) {
    arr[i]();
  }
}

// node_modules/svelte/src/internal/client/constants.js
var DERIVED = 1 << 1;
var EFFECT = 1 << 2;
var RENDER_EFFECT = 1 << 3;
var BLOCK_EFFECT = 1 << 4;
var BRANCH_EFFECT = 1 << 5;
var ROOT_EFFECT = 1 << 6;
var BOUNDARY_EFFECT = 1 << 7;
var UNOWNED = 1 << 8;
var DISCONNECTED = 1 << 9;
var CLEAN = 1 << 10;
var DIRTY = 1 << 11;
var MAYBE_DIRTY = 1 << 12;
var INERT = 1 << 13;
var DESTROYED = 1 << 14;
var EFFECT_RAN = 1 << 15;
var EFFECT_TRANSPARENT = 1 << 16;
var LEGACY_DERIVED_PROP = 1 << 17;
var INSPECT_EFFECT = 1 << 18;
var HEAD_EFFECT = 1 << 19;
var EFFECT_HAS_DERIVED = 1 << 20;
var STATE_SYMBOL = Symbol("$state");
var STATE_SYMBOL_METADATA = Symbol("$state metadata");
var LEGACY_PROPS = Symbol("legacy props");
var LOADING_ATTR_SYMBOL = Symbol("");

// node_modules/svelte/src/internal/client/reactivity/equality.js
function equals(value) {
  return value === this.v;
}
function safe_not_equal(a, b) {
  return a != a ? b == b : a !== b || a !== null && typeof a === "object" || typeof a === "function";
}
function not_equal(a, b) {
  return a !== b;
}
function safe_equals(value) {
  return !safe_not_equal(value, this.v);
}

// node_modules/svelte/src/internal/client/errors.js
function bind_invalid_checkbox_value() {
  if (dev_fallback_default) {
    const error = new Error(`bind_invalid_checkbox_value
Using \`bind:value\` together with a checkbox input is not allowed. Use \`bind:checked\` instead
https://svelte.dev/e/bind_invalid_checkbox_value`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/bind_invalid_checkbox_value`);
  }
}
function derived_references_self() {
  if (dev_fallback_default) {
    const error = new Error(`derived_references_self
A derived value cannot reference itself recursively
https://svelte.dev/e/derived_references_self`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/derived_references_self`);
  }
}
function effect_in_teardown(rune) {
  if (dev_fallback_default) {
    const error = new Error(`effect_in_teardown
\`${rune}\` cannot be used inside an effect cleanup function
https://svelte.dev/e/effect_in_teardown`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/effect_in_teardown`);
  }
}
function effect_in_unowned_derived() {
  if (dev_fallback_default) {
    const error = new Error(`effect_in_unowned_derived
Effect cannot be created inside a \`$derived\` value that was not itself created inside an effect
https://svelte.dev/e/effect_in_unowned_derived`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/effect_in_unowned_derived`);
  }
}
function effect_orphan(rune) {
  if (dev_fallback_default) {
    const error = new Error(`effect_orphan
\`${rune}\` can only be used inside an effect (e.g. during component initialisation)
https://svelte.dev/e/effect_orphan`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/effect_orphan`);
  }
}
function effect_update_depth_exceeded() {
  if (dev_fallback_default) {
    const error = new Error(`effect_update_depth_exceeded
Maximum update depth exceeded. This can happen when a reactive block or effect repeatedly sets a new value. Svelte limits the number of nested updates to prevent infinite loops
https://svelte.dev/e/effect_update_depth_exceeded`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/effect_update_depth_exceeded`);
  }
}
function hydration_failed() {
  if (dev_fallback_default) {
    const error = new Error(`hydration_failed
Failed to hydrate the application
https://svelte.dev/e/hydration_failed`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/hydration_failed`);
  }
}
function props_invalid_value(key) {
  if (dev_fallback_default) {
    const error = new Error(`props_invalid_value
Cannot do \`bind:${key}={undefined}\` when \`${key}\` has a fallback value
https://svelte.dev/e/props_invalid_value`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/props_invalid_value`);
  }
}
function rune_outside_svelte(rune) {
  if (dev_fallback_default) {
    const error = new Error(`rune_outside_svelte
The \`${rune}\` rune is only available inside \`.svelte\` and \`.svelte.js/ts\` files
https://svelte.dev/e/rune_outside_svelte`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/rune_outside_svelte`);
  }
}
function state_descriptors_fixed() {
  if (dev_fallback_default) {
    const error = new Error(`state_descriptors_fixed
Property descriptors defined on \`$state\` objects must contain \`value\` and always be \`enumerable\`, \`configurable\` and \`writable\`.
https://svelte.dev/e/state_descriptors_fixed`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/state_descriptors_fixed`);
  }
}
function state_prototype_fixed() {
  if (dev_fallback_default) {
    const error = new Error(`state_prototype_fixed
Cannot set prototype of \`$state\` object
https://svelte.dev/e/state_prototype_fixed`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/state_prototype_fixed`);
  }
}
function state_unsafe_local_read() {
  if (dev_fallback_default) {
    const error = new Error(`state_unsafe_local_read
Reading state that was created inside the same derived is forbidden. Consider using \`untrack\` to read locally created state
https://svelte.dev/e/state_unsafe_local_read`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/state_unsafe_local_read`);
  }
}
function state_unsafe_mutation() {
  if (dev_fallback_default) {
    const error = new Error(`state_unsafe_mutation
Updating state inside a derived or a template expression is forbidden. If the value should not be reactive, declare it without \`$state\`
https://svelte.dev/e/state_unsafe_mutation`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/state_unsafe_mutation`);
  }
}

// node_modules/svelte/src/internal/flags/index.js
var legacy_mode_flag = false;
function enable_legacy_mode_flag() {
  legacy_mode_flag = true;
}

// node_modules/svelte/src/internal/client/reactivity/sources.js
var inspect_effects = /* @__PURE__ */ new Set();
function set_inspect_effects(v) {
  inspect_effects = v;
}
function source(v) {
  return {
    f: 0,
    // TODO ideally we could skip this altogether, but it causes type errors
    v,
    reactions: null,
    equals,
    version: 0
  };
}
// @__NO_SIDE_EFFECTS__
function mutable_source(initial_value, immutable = false) {
  const s = source(initial_value);
  if (!immutable) {
    s.equals = safe_equals;
  }
  if (legacy_mode_flag && component_context !== null && component_context.l !== null) {
    (component_context.l.s ??= []).push(s);
  }
  return s;
}
function mutable_state(v, immutable = false) {
  return /* @__PURE__ */ push_derived_source(/* @__PURE__ */ mutable_source(v, immutable));
}
// @__NO_SIDE_EFFECTS__
function push_derived_source(source2) {
  if (active_reaction !== null && (active_reaction.f & DERIVED) !== 0) {
    if (derived_sources === null) {
      set_derived_sources([source2]);
    } else {
      derived_sources.push(source2);
    }
  }
  return source2;
}
function set(source2, value) {
  if (active_reaction !== null && is_runes() && (active_reaction.f & (DERIVED | BLOCK_EFFECT)) !== 0 && // If the source was created locally within the current derived, then
  // we allow the mutation.
  (derived_sources === null || !derived_sources.includes(source2))) {
    state_unsafe_mutation();
  }
  return internal_set(source2, value);
}
function internal_set(source2, value) {
  if (!source2.equals(value)) {
    source2.v = value;
    source2.version = increment_version();
    mark_reactions(source2, DIRTY);
    if (is_runes() && active_effect !== null && (active_effect.f & CLEAN) !== 0 && (active_effect.f & BRANCH_EFFECT) === 0) {
      if (new_deps !== null && new_deps.includes(source2)) {
        set_signal_status(active_effect, DIRTY);
        schedule_effect(active_effect);
      } else {
        if (untracked_writes === null) {
          set_untracked_writes([source2]);
        } else {
          untracked_writes.push(source2);
        }
      }
    }
    if (dev_fallback_default && inspect_effects.size > 0) {
      const inspects = Array.from(inspect_effects);
      var previously_flushing_effect = is_flushing_effect;
      set_is_flushing_effect(true);
      try {
        for (const effect2 of inspects) {
          if ((effect2.f & CLEAN) !== 0) {
            set_signal_status(effect2, MAYBE_DIRTY);
          }
          if (check_dirtiness(effect2)) {
            update_effect(effect2);
          }
        }
      } finally {
        set_is_flushing_effect(previously_flushing_effect);
      }
      inspect_effects.clear();
    }
  }
  return value;
}
function mark_reactions(signal, status) {
  var reactions = signal.reactions;
  if (reactions === null) return;
  var runes = is_runes();
  var length = reactions.length;
  for (var i = 0; i < length; i++) {
    var reaction = reactions[i];
    var flags = reaction.f;
    if ((flags & DIRTY) !== 0) continue;
    if (!runes && reaction === active_effect) continue;
    if (dev_fallback_default && (flags & INSPECT_EFFECT) !== 0) {
      inspect_effects.add(reaction);
      continue;
    }
    set_signal_status(reaction, status);
    if ((flags & (CLEAN | UNOWNED)) !== 0) {
      if ((flags & DERIVED) !== 0) {
        mark_reactions(
          /** @type {Derived} */
          reaction,
          MAYBE_DIRTY
        );
      } else {
        schedule_effect(
          /** @type {Effect} */
          reaction
        );
      }
    }
  }
}

// node_modules/svelte/src/internal/client/warnings.js
var bold = "font-weight: bold";
var normal = "font-weight: normal";
function hydration_attribute_changed(attribute, html2, value) {
  if (dev_fallback_default) {
    console.warn(`%c[svelte] hydration_attribute_changed
%cThe \`${attribute}\` attribute on \`${html2}\` changed its value between server and client renders. The client value, \`${value}\`, will be ignored in favour of the server value
https://svelte.dev/e/hydration_attribute_changed`, bold, normal);
  } else {
    console.warn(`https://svelte.dev/e/hydration_attribute_changed`);
  }
}
function hydration_mismatch(location) {
  if (dev_fallback_default) {
    console.warn(`%c[svelte] hydration_mismatch
%c${location ? `Hydration failed because the initial UI does not match what was rendered on the server. The error occurred near ${location}` : "Hydration failed because the initial UI does not match what was rendered on the server"}
https://svelte.dev/e/hydration_mismatch`, bold, normal);
  } else {
    console.warn(`https://svelte.dev/e/hydration_mismatch`);
  }
}
function lifecycle_double_unmount() {
  if (dev_fallback_default) {
    console.warn(`%c[svelte] lifecycle_double_unmount
%cTried to unmount a component that was not mounted
https://svelte.dev/e/lifecycle_double_unmount`, bold, normal);
  } else {
    console.warn(`https://svelte.dev/e/lifecycle_double_unmount`);
  }
}
function ownership_invalid_mutation(component2, owner) {
  if (dev_fallback_default) {
    console.warn(`%c[svelte] ownership_invalid_mutation
%c${component2 ? `${component2} mutated a value owned by ${owner}. This is strongly discouraged. Consider passing values to child components with \`bind:\`, or use a callback instead` : "Mutating a value outside the component that created it is strongly discouraged. Consider passing values to child components with `bind:`, or use a callback instead"}
https://svelte.dev/e/ownership_invalid_mutation`, bold, normal);
  } else {
    console.warn(`https://svelte.dev/e/ownership_invalid_mutation`);
  }
}
function state_proxy_equality_mismatch(operator) {
  if (dev_fallback_default) {
    console.warn(`%c[svelte] state_proxy_equality_mismatch
%cReactive \`$state(...)\` proxies and the values they proxy have different identities. Because of this, comparisons with \`${operator}\` will produce unexpected results
https://svelte.dev/e/state_proxy_equality_mismatch`, bold, normal);
  } else {
    console.warn(`https://svelte.dev/e/state_proxy_equality_mismatch`);
  }
}

// node_modules/svelte/src/constants.js
var EACH_ITEM_REACTIVE = 1;
var EACH_INDEX_REACTIVE = 1 << 1;
var EACH_IS_CONTROLLED = 1 << 2;
var EACH_IS_ANIMATED = 1 << 3;
var EACH_ITEM_IMMUTABLE = 1 << 4;
var PROPS_IS_IMMUTABLE = 1;
var PROPS_IS_RUNES = 1 << 1;
var PROPS_IS_UPDATED = 1 << 2;
var PROPS_IS_BINDABLE = 1 << 3;
var PROPS_IS_LAZY_INITIAL = 1 << 4;
var TRANSITION_OUT = 1 << 1;
var TRANSITION_GLOBAL = 1 << 2;
var TEMPLATE_FRAGMENT = 1;
var TEMPLATE_USE_IMPORT_NODE = 1 << 1;
var HYDRATION_START = "[";
var HYDRATION_START_ELSE = "[!";
var HYDRATION_END = "]";
var HYDRATION_ERROR = {};
var ELEMENT_PRESERVE_ATTRIBUTE_CASE = 1 << 1;
var UNINITIALIZED = Symbol();
var FILENAME = Symbol("filename");
var HMR = Symbol("hmr");

// node_modules/svelte/src/internal/client/dom/hydration.js
var hydrating = false;
function set_hydrating(value) {
  hydrating = value;
}
var hydrate_node;
function set_hydrate_node(node) {
  if (node === null) {
    hydration_mismatch();
    throw HYDRATION_ERROR;
  }
  return hydrate_node = node;
}
function hydrate_next() {
  return set_hydrate_node(
    /** @type {TemplateNode} */
    get_next_sibling(hydrate_node)
  );
}
function reset(node) {
  if (!hydrating) return;
  if (get_next_sibling(hydrate_node) !== null) {
    hydration_mismatch();
    throw HYDRATION_ERROR;
  }
  hydrate_node = node;
}
function next(count = 1) {
  if (hydrating) {
    var i = count;
    var node = hydrate_node;
    while (i--) {
      node = /** @type {TemplateNode} */
      get_next_sibling(node);
    }
    hydrate_node = node;
  }
}
function remove_nodes() {
  var depth = 0;
  var node = hydrate_node;
  while (true) {
    if (node.nodeType === 8) {
      var data = (
        /** @type {Comment} */
        node.data
      );
      if (data === HYDRATION_END) {
        if (depth === 0) return node;
        depth -= 1;
      } else if (data === HYDRATION_START || data === HYDRATION_START_ELSE) {
        depth += 1;
      }
    }
    var next2 = (
      /** @type {TemplateNode} */
      get_next_sibling(node)
    );
    node.remove();
    node = next2;
  }
}

// node_modules/svelte/src/internal/client/dev/ownership.js
var boundaries = {};
var chrome_pattern = /at (?:.+ \()?(.+):(\d+):(\d+)\)?$/;
var firefox_pattern = /@(.+):(\d+):(\d+)$/;
function get_stack() {
  const stack2 = new Error().stack;
  if (!stack2) return null;
  const entries = [];
  for (const line of stack2.split("\n")) {
    let match = chrome_pattern.exec(line) ?? firefox_pattern.exec(line);
    if (match) {
      entries.push({
        file: match[1],
        line: +match[2],
        column: +match[3]
      });
    }
  }
  return entries;
}
function get_component() {
  const stack2 = get_stack()?.slice(4);
  if (!stack2) return null;
  for (let i = 0; i < stack2.length; i++) {
    const entry = stack2[i];
    const modules = boundaries[entry.file];
    if (!modules) {
      if (i === 0) return null;
      continue;
    }
    for (const module of modules) {
      if (module.end == null) {
        return null;
      }
      if (module.start.line < entry.line && module.end.line > entry.line) {
        return module.component;
      }
    }
  }
  return null;
}
var ADD_OWNER = Symbol("ADD_OWNER");
function widen_ownership(from, to) {
  if (to.owners === null) {
    return;
  }
  while (from) {
    if (from.owners === null) {
      to.owners = null;
      break;
    }
    for (const owner of from.owners) {
      to.owners.add(owner);
    }
    from = from.parent;
  }
}
function has_owner(metadata, component2) {
  if (metadata.owners === null) {
    return true;
  }
  return metadata.owners.has(component2) || metadata.parent !== null && has_owner(metadata.parent, component2);
}
function get_owner(metadata) {
  return metadata?.owners?.values().next().value ?? get_owner(
    /** @type {ProxyMetadata} */
    metadata.parent
  );
}
var skip = false;
function check_ownership(metadata) {
  if (skip) return;
  const component2 = get_component();
  if (component2 && !has_owner(metadata, component2)) {
    let original = get_owner(metadata);
    if (original[FILENAME] !== component2[FILENAME]) {
      ownership_invalid_mutation(component2[FILENAME], original[FILENAME]);
    } else {
      ownership_invalid_mutation();
    }
  }
}

// node_modules/svelte/src/internal/client/proxy.js
function proxy(value, parent = null, prev) {
  if (typeof value !== "object" || value === null || STATE_SYMBOL in value) {
    return value;
  }
  const prototype = get_prototype_of(value);
  if (prototype !== object_prototype && prototype !== array_prototype) {
    return value;
  }
  var sources = /* @__PURE__ */ new Map();
  var is_proxied_array = is_array(value);
  var version = source(0);
  if (is_proxied_array) {
    sources.set("length", source(
      /** @type {any[]} */
      value.length
    ));
  }
  var metadata;
  if (dev_fallback_default) {
    metadata = {
      parent,
      owners: null
    };
    if (prev) {
      const prev_owners = prev.v?.[STATE_SYMBOL_METADATA]?.owners;
      metadata.owners = prev_owners ? new Set(prev_owners) : null;
    } else {
      metadata.owners = parent === null ? component_context !== null ? /* @__PURE__ */ new Set([component_context.function]) : null : /* @__PURE__ */ new Set();
    }
  }
  return new Proxy(
    /** @type {any} */
    value,
    {
      defineProperty(_, prop2, descriptor) {
        if (!("value" in descriptor) || descriptor.configurable === false || descriptor.enumerable === false || descriptor.writable === false) {
          state_descriptors_fixed();
        }
        var s = sources.get(prop2);
        if (s === void 0) {
          s = source(descriptor.value);
          sources.set(prop2, s);
        } else {
          set(s, proxy(descriptor.value, metadata));
        }
        return true;
      },
      deleteProperty(target2, prop2) {
        var s = sources.get(prop2);
        if (s === void 0) {
          if (prop2 in target2) {
            sources.set(prop2, source(UNINITIALIZED));
          }
        } else {
          if (is_proxied_array && typeof prop2 === "string") {
            var ls = (
              /** @type {Source<number>} */
              sources.get("length")
            );
            var n = Number(prop2);
            if (Number.isInteger(n) && n < ls.v) {
              set(ls, n);
            }
          }
          set(s, UNINITIALIZED);
          update_version(version);
        }
        return true;
      },
      get(target2, prop2, receiver) {
        if (dev_fallback_default && prop2 === STATE_SYMBOL_METADATA) {
          return metadata;
        }
        if (prop2 === STATE_SYMBOL) {
          return value;
        }
        var s = sources.get(prop2);
        var exists = prop2 in target2;
        if (s === void 0 && (!exists || get_descriptor(target2, prop2)?.writable)) {
          s = source(proxy(exists ? target2[prop2] : UNINITIALIZED, metadata));
          sources.set(prop2, s);
        }
        if (s !== void 0) {
          var v = get(s);
          if (dev_fallback_default) {
            var prop_metadata = v?.[STATE_SYMBOL_METADATA];
            if (prop_metadata && prop_metadata?.parent !== metadata) {
              widen_ownership(metadata, prop_metadata);
            }
          }
          return v === UNINITIALIZED ? void 0 : v;
        }
        return Reflect.get(target2, prop2, receiver);
      },
      getOwnPropertyDescriptor(target2, prop2) {
        var descriptor = Reflect.getOwnPropertyDescriptor(target2, prop2);
        if (descriptor && "value" in descriptor) {
          var s = sources.get(prop2);
          if (s) descriptor.value = get(s);
        } else if (descriptor === void 0) {
          var source2 = sources.get(prop2);
          var value2 = source2?.v;
          if (source2 !== void 0 && value2 !== UNINITIALIZED) {
            return {
              enumerable: true,
              configurable: true,
              value: value2,
              writable: true
            };
          }
        }
        return descriptor;
      },
      has(target2, prop2) {
        if (dev_fallback_default && prop2 === STATE_SYMBOL_METADATA) {
          return true;
        }
        if (prop2 === STATE_SYMBOL) {
          return true;
        }
        var s = sources.get(prop2);
        var has = s !== void 0 && s.v !== UNINITIALIZED || Reflect.has(target2, prop2);
        if (s !== void 0 || active_effect !== null && (!has || get_descriptor(target2, prop2)?.writable)) {
          if (s === void 0) {
            s = source(has ? proxy(target2[prop2], metadata) : UNINITIALIZED);
            sources.set(prop2, s);
          }
          var value2 = get(s);
          if (value2 === UNINITIALIZED) {
            return false;
          }
        }
        return has;
      },
      set(target2, prop2, value2, receiver) {
        var s = sources.get(prop2);
        var has = prop2 in target2;
        if (is_proxied_array && prop2 === "length") {
          for (var i = value2; i < /** @type {Source<number>} */
          s.v; i += 1) {
            var other_s = sources.get(i + "");
            if (other_s !== void 0) {
              set(other_s, UNINITIALIZED);
            } else if (i in target2) {
              other_s = source(UNINITIALIZED);
              sources.set(i + "", other_s);
            }
          }
        }
        if (s === void 0) {
          if (!has || get_descriptor(target2, prop2)?.writable) {
            s = source(void 0);
            set(s, proxy(value2, metadata));
            sources.set(prop2, s);
          }
        } else {
          has = s.v !== UNINITIALIZED;
          set(s, proxy(value2, metadata));
        }
        if (dev_fallback_default) {
          var prop_metadata = value2?.[STATE_SYMBOL_METADATA];
          if (prop_metadata && prop_metadata?.parent !== metadata) {
            widen_ownership(metadata, prop_metadata);
          }
          check_ownership(metadata);
        }
        var descriptor = Reflect.getOwnPropertyDescriptor(target2, prop2);
        if (descriptor?.set) {
          descriptor.set.call(receiver, value2);
        }
        if (!has) {
          if (is_proxied_array && typeof prop2 === "string") {
            var ls = (
              /** @type {Source<number>} */
              sources.get("length")
            );
            var n = Number(prop2);
            if (Number.isInteger(n) && n >= ls.v) {
              set(ls, n + 1);
            }
          }
          update_version(version);
        }
        return true;
      },
      ownKeys(target2) {
        get(version);
        var own_keys = Reflect.ownKeys(target2).filter((key2) => {
          var source3 = sources.get(key2);
          return source3 === void 0 || source3.v !== UNINITIALIZED;
        });
        for (var [key, source2] of sources) {
          if (source2.v !== UNINITIALIZED && !(key in target2)) {
            own_keys.push(key);
          }
        }
        return own_keys;
      },
      setPrototypeOf() {
        state_prototype_fixed();
      }
    }
  );
}
function update_version(signal, d = 1) {
  set(signal, signal.v + d);
}
function get_proxied_value(value) {
  if (value !== null && typeof value === "object" && STATE_SYMBOL in value) {
    return value[STATE_SYMBOL];
  }
  return value;
}
function is(a, b) {
  return Object.is(get_proxied_value(a), get_proxied_value(b));
}

// node_modules/svelte/src/internal/client/dev/equality.js
function init_array_prototype_warnings() {
  const array_prototype2 = Array.prototype;
  const cleanup = Array.__svelte_cleanup;
  if (cleanup) {
    cleanup();
  }
  const { indexOf, lastIndexOf, includes } = array_prototype2;
  array_prototype2.indexOf = function(item, from_index) {
    const index2 = indexOf.call(this, item, from_index);
    if (index2 === -1) {
      const test = indexOf.call(get_proxied_value(this), get_proxied_value(item), from_index);
      if (test !== -1) {
        state_proxy_equality_mismatch("array.indexOf(...)");
      }
    }
    return index2;
  };
  array_prototype2.lastIndexOf = function(item, from_index) {
    const index2 = lastIndexOf.call(this, item, from_index ?? this.length - 1);
    if (index2 === -1) {
      const test = lastIndexOf.call(
        get_proxied_value(this),
        get_proxied_value(item),
        from_index ?? this.length - 1
      );
      if (test !== -1) {
        state_proxy_equality_mismatch("array.lastIndexOf(...)");
      }
    }
    return index2;
  };
  array_prototype2.includes = function(item, from_index) {
    const has = includes.call(this, item, from_index);
    if (!has) {
      const test = includes.call(get_proxied_value(this), get_proxied_value(item), from_index);
      if (test) {
        state_proxy_equality_mismatch("array.includes(...)");
      }
    }
    return has;
  };
  Array.__svelte_cleanup = () => {
    array_prototype2.indexOf = indexOf;
    array_prototype2.lastIndexOf = lastIndexOf;
    array_prototype2.includes = includes;
  };
}

// node_modules/svelte/src/internal/client/dom/operations.js
var $window;
var $document;
var first_child_getter;
var next_sibling_getter;
function init_operations() {
  if ($window !== void 0) {
    return;
  }
  $window = window;
  $document = document;
  var element_prototype = Element.prototype;
  var node_prototype = Node.prototype;
  first_child_getter = get_descriptor(node_prototype, "firstChild").get;
  next_sibling_getter = get_descriptor(node_prototype, "nextSibling").get;
  element_prototype.__click = void 0;
  element_prototype.__className = "";
  element_prototype.__attributes = null;
  element_prototype.__styles = null;
  element_prototype.__e = void 0;
  Text.prototype.__t = void 0;
  if (dev_fallback_default) {
    element_prototype.__svelte_meta = null;
    init_array_prototype_warnings();
  }
}
function create_text(value = "") {
  return document.createTextNode(value);
}
// @__NO_SIDE_EFFECTS__
function get_first_child(node) {
  return first_child_getter.call(node);
}
// @__NO_SIDE_EFFECTS__
function get_next_sibling(node) {
  return next_sibling_getter.call(node);
}
function child(node, is_text) {
  if (!hydrating) {
    return /* @__PURE__ */ get_first_child(node);
  }
  var child2 = (
    /** @type {TemplateNode} */
    /* @__PURE__ */ get_first_child(hydrate_node)
  );
  if (child2 === null) {
    child2 = hydrate_node.appendChild(create_text());
  } else if (is_text && child2.nodeType !== 3) {
    var text2 = create_text();
    child2?.before(text2);
    set_hydrate_node(text2);
    return text2;
  }
  set_hydrate_node(child2);
  return child2;
}
function first_child(fragment, is_text) {
  if (!hydrating) {
    var first = (
      /** @type {DocumentFragment} */
      /* @__PURE__ */ get_first_child(
        /** @type {Node} */
        fragment
      )
    );
    if (first instanceof Comment && first.data === "") return /* @__PURE__ */ get_next_sibling(first);
    return first;
  }
  if (is_text && hydrate_node?.nodeType !== 3) {
    var text2 = create_text();
    hydrate_node?.before(text2);
    set_hydrate_node(text2);
    return text2;
  }
  return hydrate_node;
}
function sibling(node, count = 1, is_text = false) {
  let next_sibling = hydrating ? hydrate_node : node;
  var last_sibling;
  while (count--) {
    last_sibling = next_sibling;
    next_sibling = /** @type {TemplateNode} */
    /* @__PURE__ */ get_next_sibling(next_sibling);
  }
  if (!hydrating) {
    return next_sibling;
  }
  var type = next_sibling?.nodeType;
  if (is_text && type !== 3) {
    var text2 = create_text();
    if (next_sibling === null) {
      last_sibling?.after(text2);
    } else {
      next_sibling.before(text2);
    }
    set_hydrate_node(text2);
    return text2;
  }
  set_hydrate_node(next_sibling);
  return (
    /** @type {TemplateNode} */
    next_sibling
  );
}
function clear_text_content(node) {
  node.textContent = "";
}

// node_modules/svelte/src/internal/client/reactivity/deriveds.js
// @__NO_SIDE_EFFECTS__
function derived(fn) {
  var flags = DERIVED | DIRTY;
  if (active_effect === null) {
    flags |= UNOWNED;
  } else {
    active_effect.f |= EFFECT_HAS_DERIVED;
  }
  var parent_derived = active_reaction !== null && (active_reaction.f & DERIVED) !== 0 ? (
    /** @type {Derived} */
    active_reaction
  ) : null;
  const signal = {
    children: null,
    ctx: component_context,
    deps: null,
    equals,
    f: flags,
    fn,
    reactions: null,
    v: (
      /** @type {V} */
      null
    ),
    version: 0,
    parent: parent_derived ?? active_effect
  };
  if (parent_derived !== null) {
    (parent_derived.children ??= []).push(signal);
  }
  return signal;
}
// @__NO_SIDE_EFFECTS__
function derived_safe_equal(fn) {
  const signal = /* @__PURE__ */ derived(fn);
  signal.equals = safe_equals;
  return signal;
}
function destroy_derived_children(derived3) {
  var children = derived3.children;
  if (children !== null) {
    derived3.children = null;
    for (var i = 0; i < children.length; i += 1) {
      var child2 = children[i];
      if ((child2.f & DERIVED) !== 0) {
        destroy_derived(
          /** @type {Derived} */
          child2
        );
      } else {
        destroy_effect(
          /** @type {Effect} */
          child2
        );
      }
    }
  }
}
var stack = [];
function get_derived_parent_effect(derived3) {
  var parent = derived3.parent;
  while (parent !== null) {
    if ((parent.f & DERIVED) === 0) {
      return (
        /** @type {Effect} */
        parent
      );
    }
    parent = parent.parent;
  }
  return null;
}
function execute_derived(derived3) {
  var value;
  var prev_active_effect = active_effect;
  set_active_effect(get_derived_parent_effect(derived3));
  if (dev_fallback_default) {
    let prev_inspect_effects = inspect_effects;
    set_inspect_effects(/* @__PURE__ */ new Set());
    try {
      if (stack.includes(derived3)) {
        derived_references_self();
      }
      stack.push(derived3);
      destroy_derived_children(derived3);
      value = update_reaction(derived3);
    } finally {
      set_active_effect(prev_active_effect);
      set_inspect_effects(prev_inspect_effects);
      stack.pop();
    }
  } else {
    try {
      destroy_derived_children(derived3);
      value = update_reaction(derived3);
    } finally {
      set_active_effect(prev_active_effect);
    }
  }
  return value;
}
function update_derived(derived3) {
  var value = execute_derived(derived3);
  var status = (skip_reaction || (derived3.f & UNOWNED) !== 0) && derived3.deps !== null ? MAYBE_DIRTY : CLEAN;
  set_signal_status(derived3, status);
  if (!derived3.equals(value)) {
    derived3.v = value;
    derived3.version = increment_version();
  }
}
function destroy_derived(derived3) {
  destroy_derived_children(derived3);
  remove_reactions(derived3, 0);
  set_signal_status(derived3, DESTROYED);
  derived3.v = derived3.children = derived3.deps = derived3.ctx = derived3.reactions = null;
}

// node_modules/svelte/src/internal/client/reactivity/effects.js
function validate_effect(rune) {
  if (active_effect === null && active_reaction === null) {
    effect_orphan(rune);
  }
  if (active_reaction !== null && (active_reaction.f & UNOWNED) !== 0) {
    effect_in_unowned_derived();
  }
  if (is_destroying_effect) {
    effect_in_teardown(rune);
  }
}
function push_effect(effect2, parent_effect) {
  var parent_last = parent_effect.last;
  if (parent_last === null) {
    parent_effect.last = parent_effect.first = effect2;
  } else {
    parent_last.next = effect2;
    effect2.prev = parent_last;
    parent_effect.last = effect2;
  }
}
function create_effect(type, fn, sync, push2 = true) {
  var is_root = (type & ROOT_EFFECT) !== 0;
  var parent_effect = active_effect;
  if (dev_fallback_default) {
    while (parent_effect !== null && (parent_effect.f & INSPECT_EFFECT) !== 0) {
      parent_effect = parent_effect.parent;
    }
  }
  var effect2 = {
    ctx: component_context,
    deps: null,
    deriveds: null,
    nodes_start: null,
    nodes_end: null,
    f: type | DIRTY,
    first: null,
    fn,
    last: null,
    next: null,
    parent: is_root ? null : parent_effect,
    prev: null,
    teardown: null,
    transitions: null,
    version: 0
  };
  if (dev_fallback_default) {
    effect2.component_function = dev_current_component_function;
  }
  if (sync) {
    var previously_flushing_effect = is_flushing_effect;
    try {
      set_is_flushing_effect(true);
      update_effect(effect2);
      effect2.f |= EFFECT_RAN;
    } catch (e) {
      destroy_effect(effect2);
      throw e;
    } finally {
      set_is_flushing_effect(previously_flushing_effect);
    }
  } else if (fn !== null) {
    schedule_effect(effect2);
  }
  var inert = sync && effect2.deps === null && effect2.first === null && effect2.nodes_start === null && effect2.teardown === null && (effect2.f & EFFECT_HAS_DERIVED) === 0;
  if (!inert && !is_root && push2) {
    if (parent_effect !== null) {
      push_effect(effect2, parent_effect);
    }
    if (active_reaction !== null && (active_reaction.f & DERIVED) !== 0) {
      var derived3 = (
        /** @type {Derived} */
        active_reaction
      );
      (derived3.children ??= []).push(effect2);
    }
  }
  return effect2;
}
function teardown(fn) {
  const effect2 = create_effect(RENDER_EFFECT, null, false);
  set_signal_status(effect2, CLEAN);
  effect2.teardown = fn;
  return effect2;
}
function user_effect(fn) {
  validate_effect("$effect");
  var defer = active_effect !== null && (active_effect.f & BRANCH_EFFECT) !== 0 && component_context !== null && !component_context.m;
  if (dev_fallback_default) {
    define_property(fn, "name", {
      value: "$effect"
    });
  }
  if (defer) {
    var context = (
      /** @type {ComponentContext} */
      component_context
    );
    (context.e ??= []).push({
      fn,
      effect: active_effect,
      reaction: active_reaction
    });
  } else {
    var signal = effect(fn);
    return signal;
  }
}
function user_pre_effect(fn) {
  validate_effect("$effect.pre");
  if (dev_fallback_default) {
    define_property(fn, "name", {
      value: "$effect.pre"
    });
  }
  return render_effect(fn);
}
function effect_root(fn) {
  const effect2 = create_effect(ROOT_EFFECT, fn, true);
  return () => {
    destroy_effect(effect2);
  };
}
function effect(fn) {
  return create_effect(EFFECT, fn, false);
}
function legacy_pre_effect(deps, fn) {
  var context = (
    /** @type {ComponentContextLegacy} */
    component_context
  );
  var token = { effect: null, ran: false };
  context.l.r1.push(token);
  token.effect = render_effect(() => {
    deps();
    if (token.ran) return;
    token.ran = true;
    set(context.l.r2, true);
    untrack(fn);
  });
}
function legacy_pre_effect_reset() {
  var context = (
    /** @type {ComponentContextLegacy} */
    component_context
  );
  render_effect(() => {
    if (!get(context.l.r2)) return;
    for (var token of context.l.r1) {
      var effect2 = token.effect;
      if ((effect2.f & CLEAN) !== 0) {
        set_signal_status(effect2, MAYBE_DIRTY);
      }
      if (check_dirtiness(effect2)) {
        update_effect(effect2);
      }
      token.ran = false;
    }
    context.l.r2.v = false;
  });
}
function render_effect(fn) {
  return create_effect(RENDER_EFFECT, fn, true);
}
function template_effect(fn) {
  if (dev_fallback_default) {
    define_property(fn, "name", {
      value: "{expression}"
    });
  }
  return block(fn);
}
function block(fn, flags = 0) {
  return create_effect(RENDER_EFFECT | BLOCK_EFFECT | flags, fn, true);
}
function branch(fn, push2 = true) {
  return create_effect(RENDER_EFFECT | BRANCH_EFFECT, fn, true, push2);
}
function execute_effect_teardown(effect2) {
  var teardown2 = effect2.teardown;
  if (teardown2 !== null) {
    const previously_destroying_effect = is_destroying_effect;
    const previous_reaction = active_reaction;
    set_is_destroying_effect(true);
    set_active_reaction(null);
    try {
      teardown2.call(null);
    } finally {
      set_is_destroying_effect(previously_destroying_effect);
      set_active_reaction(previous_reaction);
    }
  }
}
function destroy_effect_deriveds(signal) {
  var deriveds = signal.deriveds;
  if (deriveds !== null) {
    signal.deriveds = null;
    for (var i = 0; i < deriveds.length; i += 1) {
      destroy_derived(deriveds[i]);
    }
  }
}
function destroy_effect_children(signal, remove_dom = false) {
  var effect2 = signal.first;
  signal.first = signal.last = null;
  while (effect2 !== null) {
    var next2 = effect2.next;
    destroy_effect(effect2, remove_dom);
    effect2 = next2;
  }
}
function destroy_block_effect_children(signal) {
  var effect2 = signal.first;
  while (effect2 !== null) {
    var next2 = effect2.next;
    if ((effect2.f & BRANCH_EFFECT) === 0) {
      destroy_effect(effect2);
    }
    effect2 = next2;
  }
}
function destroy_effect(effect2, remove_dom = true) {
  var removed = false;
  if ((remove_dom || (effect2.f & HEAD_EFFECT) !== 0) && effect2.nodes_start !== null) {
    var node = effect2.nodes_start;
    var end = effect2.nodes_end;
    while (node !== null) {
      var next2 = node === end ? null : (
        /** @type {TemplateNode} */
        get_next_sibling(node)
      );
      node.remove();
      node = next2;
    }
    removed = true;
  }
  destroy_effect_children(effect2, remove_dom && !removed);
  destroy_effect_deriveds(effect2);
  remove_reactions(effect2, 0);
  set_signal_status(effect2, DESTROYED);
  var transitions = effect2.transitions;
  if (transitions !== null) {
    for (const transition2 of transitions) {
      transition2.stop();
    }
  }
  execute_effect_teardown(effect2);
  var parent = effect2.parent;
  if (parent !== null && parent.first !== null) {
    unlink_effect(effect2);
  }
  if (dev_fallback_default) {
    effect2.component_function = null;
  }
  effect2.next = effect2.prev = effect2.teardown = effect2.ctx = effect2.deps = effect2.fn = effect2.nodes_start = effect2.nodes_end = null;
}
function unlink_effect(effect2) {
  var parent = effect2.parent;
  var prev = effect2.prev;
  var next2 = effect2.next;
  if (prev !== null) prev.next = next2;
  if (next2 !== null) next2.prev = prev;
  if (parent !== null) {
    if (parent.first === effect2) parent.first = next2;
    if (parent.last === effect2) parent.last = prev;
  }
}
function pause_effect(effect2, callback) {
  var transitions = [];
  pause_children(effect2, transitions, true);
  run_out_transitions(transitions, () => {
    destroy_effect(effect2);
    if (callback) callback();
  });
}
function run_out_transitions(transitions, fn) {
  var remaining = transitions.length;
  if (remaining > 0) {
    var check = () => --remaining || fn();
    for (var transition2 of transitions) {
      transition2.out(check);
    }
  } else {
    fn();
  }
}
function pause_children(effect2, transitions, local) {
  if ((effect2.f & INERT) !== 0) return;
  effect2.f ^= INERT;
  if (effect2.transitions !== null) {
    for (const transition2 of effect2.transitions) {
      if (transition2.is_global || local) {
        transitions.push(transition2);
      }
    }
  }
  var child2 = effect2.first;
  while (child2 !== null) {
    var sibling2 = child2.next;
    var transparent = (child2.f & EFFECT_TRANSPARENT) !== 0 || (child2.f & BRANCH_EFFECT) !== 0;
    pause_children(child2, transitions, transparent ? local : false);
    child2 = sibling2;
  }
}
function resume_effect(effect2) {
  resume_children(effect2, true);
}
function resume_children(effect2, local) {
  if ((effect2.f & INERT) === 0) return;
  if (check_dirtiness(effect2)) {
    update_effect(effect2);
  }
  effect2.f ^= INERT;
  var child2 = effect2.first;
  while (child2 !== null) {
    var sibling2 = child2.next;
    var transparent = (child2.f & EFFECT_TRANSPARENT) !== 0 || (child2.f & BRANCH_EFFECT) !== 0;
    resume_children(child2, transparent ? local : false);
    child2 = sibling2;
  }
  if (effect2.transitions !== null) {
    for (const transition2 of effect2.transitions) {
      if (transition2.is_global || local) {
        transition2.in();
      }
    }
  }
}

// node_modules/svelte/src/internal/client/dom/task.js
var request_idle_callback = typeof requestIdleCallback === "undefined" ? (cb) => setTimeout(cb, 1) : requestIdleCallback;
var is_micro_task_queued = false;
var is_idle_task_queued = false;
var current_queued_micro_tasks = [];
var current_queued_idle_tasks = [];
function process_micro_tasks() {
  is_micro_task_queued = false;
  const tasks = current_queued_micro_tasks.slice();
  current_queued_micro_tasks = [];
  run_all(tasks);
}
function process_idle_tasks() {
  is_idle_task_queued = false;
  const tasks = current_queued_idle_tasks.slice();
  current_queued_idle_tasks = [];
  run_all(tasks);
}
function queue_micro_task(fn) {
  if (!is_micro_task_queued) {
    is_micro_task_queued = true;
    queueMicrotask(process_micro_tasks);
  }
  current_queued_micro_tasks.push(fn);
}
function queue_idle_task(fn) {
  if (!is_idle_task_queued) {
    is_idle_task_queued = true;
    request_idle_callback(process_idle_tasks);
  }
  current_queued_idle_tasks.push(fn);
}
function flush_tasks() {
  if (is_micro_task_queued) {
    process_micro_tasks();
  }
  if (is_idle_task_queued) {
    process_idle_tasks();
  }
}

// node_modules/svelte/src/internal/shared/errors.js
function lifecycle_outside_component(name) {
  if (dev_fallback_default) {
    const error = new Error(`lifecycle_outside_component
\`${name}(...)\` can only be used during component initialisation
https://svelte.dev/e/lifecycle_outside_component`);
    error.name = "Svelte error";
    throw error;
  } else {
    throw new Error(`https://svelte.dev/e/lifecycle_outside_component`);
  }
}

// node_modules/svelte/src/internal/client/runtime.js
var FLUSH_MICROTASK = 0;
var FLUSH_SYNC = 1;
var handled_errors = /* @__PURE__ */ new WeakSet();
var is_throwing_error = false;
var scheduler_mode = FLUSH_MICROTASK;
var is_micro_task_queued2 = false;
var last_scheduled_effect = null;
var is_flushing_effect = false;
var is_destroying_effect = false;
function set_is_flushing_effect(value) {
  is_flushing_effect = value;
}
function set_is_destroying_effect(value) {
  is_destroying_effect = value;
}
var queued_root_effects = [];
var flush_count = 0;
var dev_effect_stack = [];
var active_reaction = null;
function set_active_reaction(reaction) {
  active_reaction = reaction;
}
var active_effect = null;
function set_active_effect(effect2) {
  active_effect = effect2;
}
var derived_sources = null;
function set_derived_sources(sources) {
  derived_sources = sources;
}
var new_deps = null;
var skipped_deps = 0;
var untracked_writes = null;
function set_untracked_writes(value) {
  untracked_writes = value;
}
var current_version = 0;
var skip_reaction = false;
var captured_signals = null;
var component_context = null;
var dev_current_component_function = null;
function increment_version() {
  return ++current_version;
}
function is_runes() {
  return !legacy_mode_flag || component_context !== null && component_context.l === null;
}
function check_dirtiness(reaction) {
  var flags = reaction.f;
  if ((flags & DIRTY) !== 0) {
    return true;
  }
  if ((flags & MAYBE_DIRTY) !== 0) {
    var dependencies = reaction.deps;
    var is_unowned = (flags & UNOWNED) !== 0;
    if (dependencies !== null) {
      var i;
      if ((flags & DISCONNECTED) !== 0) {
        for (i = 0; i < dependencies.length; i++) {
          (dependencies[i].reactions ??= []).push(reaction);
        }
        reaction.f ^= DISCONNECTED;
      }
      for (i = 0; i < dependencies.length; i++) {
        var dependency = dependencies[i];
        if (check_dirtiness(
          /** @type {Derived} */
          dependency
        )) {
          update_derived(
            /** @type {Derived} */
            dependency
          );
        }
        if (is_unowned && active_effect !== null && !skip_reaction && !dependency?.reactions?.includes(reaction)) {
          (dependency.reactions ??= []).push(reaction);
        }
        if (dependency.version > reaction.version) {
          return true;
        }
      }
    }
    if (!is_unowned) {
      set_signal_status(reaction, CLEAN);
    }
  }
  return false;
}
function propagate_error(error, effect2) {
  var current = effect2;
  while (current !== null) {
    if ((current.f & BOUNDARY_EFFECT) !== 0) {
      try {
        current.fn(error);
        return;
      } catch {
        current.f ^= BOUNDARY_EFFECT;
      }
    }
    current = current.parent;
  }
  is_throwing_error = false;
  throw error;
}
function should_rethrow_error(effect2) {
  return (effect2.f & DESTROYED) === 0 && (effect2.parent === null || (effect2.parent.f & BOUNDARY_EFFECT) === 0);
}
function handle_error(error, effect2, previous_effect, component_context2) {
  if (is_throwing_error) {
    if (previous_effect === null) {
      is_throwing_error = false;
    }
    if (should_rethrow_error(effect2)) {
      throw error;
    }
    return;
  }
  if (previous_effect !== null) {
    is_throwing_error = true;
  }
  if (!dev_fallback_default || component_context2 === null || !(error instanceof Error) || handled_errors.has(error)) {
    propagate_error(error, effect2);
    return;
  }
  handled_errors.add(error);
  const component_stack = [];
  const effect_name = effect2.fn?.name;
  if (effect_name) {
    component_stack.push(effect_name);
  }
  let current_context = component_context2;
  while (current_context !== null) {
    if (dev_fallback_default) {
      var filename = current_context.function?.[FILENAME];
      if (filename) {
        const file = filename.split("/").pop();
        component_stack.push(file);
      }
    }
    current_context = current_context.p;
  }
  const indent = /Firefox/.test(navigator.userAgent) ? "  " : "	";
  define_property(error, "message", {
    value: error.message + `
${component_stack.map((name) => `
${indent}in ${name}`).join("")}
`
  });
  define_property(error, "component_stack", {
    value: component_stack
  });
  const stack2 = error.stack;
  if (stack2) {
    const lines = stack2.split("\n");
    const new_lines = [];
    for (let i = 0; i < lines.length; i++) {
      const line = lines[i];
      if (line.includes("svelte/src/internal")) {
        continue;
      }
      new_lines.push(line);
    }
    define_property(error, "stack", {
      value: error.stack + new_lines.join("\n")
    });
  }
  propagate_error(error, effect2);
  if (should_rethrow_error(effect2)) {
    throw error;
  }
}
function update_reaction(reaction) {
  var previous_deps = new_deps;
  var previous_skipped_deps = skipped_deps;
  var previous_untracked_writes = untracked_writes;
  var previous_reaction = active_reaction;
  var previous_skip_reaction = skip_reaction;
  var prev_derived_sources = derived_sources;
  var previous_component_context = component_context;
  var flags = reaction.f;
  new_deps = /** @type {null | Value[]} */
  null;
  skipped_deps = 0;
  untracked_writes = null;
  active_reaction = (flags & (BRANCH_EFFECT | ROOT_EFFECT)) === 0 ? reaction : null;
  skip_reaction = !is_flushing_effect && (flags & UNOWNED) !== 0;
  derived_sources = null;
  component_context = reaction.ctx;
  try {
    var result = (
      /** @type {Function} */
      (0, reaction.fn)()
    );
    var deps = reaction.deps;
    if (new_deps !== null) {
      var i;
      remove_reactions(reaction, skipped_deps);
      if (deps !== null && skipped_deps > 0) {
        deps.length = skipped_deps + new_deps.length;
        for (i = 0; i < new_deps.length; i++) {
          deps[skipped_deps + i] = new_deps[i];
        }
      } else {
        reaction.deps = deps = new_deps;
      }
      if (!skip_reaction) {
        for (i = skipped_deps; i < deps.length; i++) {
          (deps[i].reactions ??= []).push(reaction);
        }
      }
    } else if (deps !== null && skipped_deps < deps.length) {
      remove_reactions(reaction, skipped_deps);
      deps.length = skipped_deps;
    }
    return result;
  } finally {
    new_deps = previous_deps;
    skipped_deps = previous_skipped_deps;
    untracked_writes = previous_untracked_writes;
    active_reaction = previous_reaction;
    skip_reaction = previous_skip_reaction;
    derived_sources = prev_derived_sources;
    component_context = previous_component_context;
  }
}
function remove_reaction(signal, dependency) {
  let reactions = dependency.reactions;
  if (reactions !== null) {
    var index2 = reactions.indexOf(signal);
    if (index2 !== -1) {
      var new_length = reactions.length - 1;
      if (new_length === 0) {
        reactions = dependency.reactions = null;
      } else {
        reactions[index2] = reactions[new_length];
        reactions.pop();
      }
    }
  }
  if (reactions === null && (dependency.f & DERIVED) !== 0 && // Destroying a child effect while updating a parent effect can cause a dependency to appear
  // to be unused, when in fact it is used by the currently-updating parent. Checking `new_deps`
  // allows us to skip the expensive work of disconnecting and immediately reconnecting it
  (new_deps === null || !new_deps.includes(dependency))) {
    set_signal_status(dependency, MAYBE_DIRTY);
    if ((dependency.f & (UNOWNED | DISCONNECTED)) === 0) {
      dependency.f ^= DISCONNECTED;
    }
    remove_reactions(
      /** @type {Derived} **/
      dependency,
      0
    );
  }
}
function remove_reactions(signal, start_index) {
  var dependencies = signal.deps;
  if (dependencies === null) return;
  for (var i = start_index; i < dependencies.length; i++) {
    remove_reaction(signal, dependencies[i]);
  }
}
function update_effect(effect2) {
  var flags = effect2.f;
  if ((flags & DESTROYED) !== 0) {
    return;
  }
  set_signal_status(effect2, CLEAN);
  var previous_effect = active_effect;
  var previous_component_context = component_context;
  active_effect = effect2;
  if (dev_fallback_default) {
    var previous_component_fn = dev_current_component_function;
    dev_current_component_function = effect2.component_function;
  }
  try {
    if ((flags & BLOCK_EFFECT) !== 0) {
      destroy_block_effect_children(effect2);
    } else {
      destroy_effect_children(effect2);
    }
    destroy_effect_deriveds(effect2);
    execute_effect_teardown(effect2);
    var teardown2 = update_reaction(effect2);
    effect2.teardown = typeof teardown2 === "function" ? teardown2 : null;
    effect2.version = current_version;
    if (dev_fallback_default) {
      dev_effect_stack.push(effect2);
    }
  } catch (error) {
    handle_error(error, effect2, previous_effect, previous_component_context || effect2.ctx);
  } finally {
    active_effect = previous_effect;
    if (dev_fallback_default) {
      dev_current_component_function = previous_component_fn;
    }
  }
}
function log_effect_stack() {
  console.error(
    "Last ten effects were: ",
    dev_effect_stack.slice(-10).map((d) => d.fn)
  );
  dev_effect_stack = [];
}
function infinite_loop_guard() {
  if (flush_count > 1e3) {
    flush_count = 0;
    try {
      effect_update_depth_exceeded();
    } catch (error) {
      if (dev_fallback_default) {
        define_property(error, "stack", {
          value: ""
        });
      }
      if (last_scheduled_effect !== null) {
        if (dev_fallback_default) {
          try {
            handle_error(error, last_scheduled_effect, null, null);
          } catch (e) {
            log_effect_stack();
            throw e;
          }
        } else {
          handle_error(error, last_scheduled_effect, null, null);
        }
      } else {
        if (dev_fallback_default) {
          log_effect_stack();
        }
        throw error;
      }
    }
  }
  flush_count++;
}
function flush_queued_root_effects(root_effects) {
  var length = root_effects.length;
  if (length === 0) {
    return;
  }
  infinite_loop_guard();
  var previously_flushing_effect = is_flushing_effect;
  is_flushing_effect = true;
  try {
    for (var i = 0; i < length; i++) {
      var effect2 = root_effects[i];
      if ((effect2.f & CLEAN) === 0) {
        effect2.f ^= CLEAN;
      }
      var collected_effects = [];
      process_effects(effect2, collected_effects);
      flush_queued_effects(collected_effects);
    }
  } finally {
    is_flushing_effect = previously_flushing_effect;
  }
}
function flush_queued_effects(effects) {
  var length = effects.length;
  if (length === 0) return;
  for (var i = 0; i < length; i++) {
    var effect2 = effects[i];
    if ((effect2.f & (DESTROYED | INERT)) === 0) {
      try {
        if (check_dirtiness(effect2)) {
          update_effect(effect2);
          if (effect2.deps === null && effect2.first === null && effect2.nodes_start === null) {
            if (effect2.teardown === null) {
              unlink_effect(effect2);
            } else {
              effect2.fn = null;
            }
          }
        }
      } catch (error) {
        handle_error(error, effect2, null, effect2.ctx);
      }
    }
  }
}
function process_deferred() {
  is_micro_task_queued2 = false;
  if (flush_count > 1001) {
    return;
  }
  const previous_queued_root_effects = queued_root_effects;
  queued_root_effects = [];
  flush_queued_root_effects(previous_queued_root_effects);
  if (!is_micro_task_queued2) {
    flush_count = 0;
    last_scheduled_effect = null;
    if (dev_fallback_default) {
      dev_effect_stack = [];
    }
  }
}
function schedule_effect(signal) {
  if (scheduler_mode === FLUSH_MICROTASK) {
    if (!is_micro_task_queued2) {
      is_micro_task_queued2 = true;
      queueMicrotask(process_deferred);
    }
  }
  last_scheduled_effect = signal;
  var effect2 = signal;
  while (effect2.parent !== null) {
    effect2 = effect2.parent;
    var flags = effect2.f;
    if ((flags & (ROOT_EFFECT | BRANCH_EFFECT)) !== 0) {
      if ((flags & CLEAN) === 0) return;
      effect2.f ^= CLEAN;
    }
  }
  queued_root_effects.push(effect2);
}
function process_effects(effect2, collected_effects) {
  var current_effect = effect2.first;
  var effects = [];
  main_loop: while (current_effect !== null) {
    var flags = current_effect.f;
    var is_branch = (flags & BRANCH_EFFECT) !== 0;
    var is_skippable_branch = is_branch && (flags & CLEAN) !== 0;
    var sibling2 = current_effect.next;
    if (!is_skippable_branch && (flags & INERT) === 0) {
      if ((flags & RENDER_EFFECT) !== 0) {
        if (is_branch) {
          current_effect.f ^= CLEAN;
        } else {
          try {
            if (check_dirtiness(current_effect)) {
              update_effect(current_effect);
            }
          } catch (error) {
            handle_error(error, current_effect, null, current_effect.ctx);
          }
        }
        var child2 = current_effect.first;
        if (child2 !== null) {
          current_effect = child2;
          continue;
        }
      } else if ((flags & EFFECT) !== 0) {
        effects.push(current_effect);
      }
    }
    if (sibling2 === null) {
      let parent = current_effect.parent;
      while (parent !== null) {
        if (effect2 === parent) {
          break main_loop;
        }
        var parent_sibling = parent.next;
        if (parent_sibling !== null) {
          current_effect = parent_sibling;
          continue main_loop;
        }
        parent = parent.parent;
      }
    }
    current_effect = sibling2;
  }
  for (var i = 0; i < effects.length; i++) {
    child2 = effects[i];
    collected_effects.push(child2);
    process_effects(child2, collected_effects);
  }
}
function flush_sync(fn) {
  var previous_scheduler_mode = scheduler_mode;
  var previous_queued_root_effects = queued_root_effects;
  try {
    infinite_loop_guard();
    const root_effects = [];
    scheduler_mode = FLUSH_SYNC;
    queued_root_effects = root_effects;
    is_micro_task_queued2 = false;
    flush_queued_root_effects(previous_queued_root_effects);
    var result = fn?.();
    flush_tasks();
    if (queued_root_effects.length > 0 || root_effects.length > 0) {
      flush_sync();
    }
    flush_count = 0;
    last_scheduled_effect = null;
    if (dev_fallback_default) {
      dev_effect_stack = [];
    }
    return result;
  } finally {
    scheduler_mode = previous_scheduler_mode;
    queued_root_effects = previous_queued_root_effects;
  }
}
function get(signal) {
  var flags = signal.f;
  var is_derived = (flags & DERIVED) !== 0;
  if (is_derived && (flags & DESTROYED) !== 0) {
    var value = execute_derived(
      /** @type {Derived} */
      signal
    );
    destroy_derived(
      /** @type {Derived} */
      signal
    );
    return value;
  }
  if (captured_signals !== null) {
    captured_signals.add(signal);
  }
  if (active_reaction !== null) {
    if (derived_sources !== null && derived_sources.includes(signal)) {
      state_unsafe_local_read();
    }
    var deps = active_reaction.deps;
    if (new_deps === null && deps !== null && deps[skipped_deps] === signal) {
      skipped_deps++;
    } else if (new_deps === null) {
      new_deps = [signal];
    } else {
      new_deps.push(signal);
    }
    if (untracked_writes !== null && active_effect !== null && (active_effect.f & CLEAN) !== 0 && (active_effect.f & BRANCH_EFFECT) === 0 && untracked_writes.includes(signal)) {
      set_signal_status(active_effect, DIRTY);
      schedule_effect(active_effect);
    }
  } else if (is_derived && /** @type {Derived} */
  signal.deps === null) {
    var derived3 = (
      /** @type {Derived} */
      signal
    );
    var parent = derived3.parent;
    var target2 = derived3;
    while (parent !== null) {
      if ((parent.f & DERIVED) !== 0) {
        var parent_derived = (
          /** @type {Derived} */
          parent
        );
        target2 = parent_derived;
        parent = parent_derived.parent;
      } else {
        var parent_effect = (
          /** @type {Effect} */
          parent
        );
        if (!parent_effect.deriveds?.includes(target2)) {
          (parent_effect.deriveds ??= []).push(target2);
        }
        break;
      }
    }
  }
  if (is_derived) {
    derived3 = /** @type {Derived} */
    signal;
    if (check_dirtiness(derived3)) {
      update_derived(derived3);
    }
  }
  return signal.v;
}
function capture_signals(fn) {
  var previous_captured_signals = captured_signals;
  captured_signals = /* @__PURE__ */ new Set();
  var captured = captured_signals;
  var signal;
  try {
    untrack(fn);
    if (previous_captured_signals !== null) {
      for (signal of captured_signals) {
        previous_captured_signals.add(signal);
      }
    }
  } finally {
    captured_signals = previous_captured_signals;
  }
  return captured;
}
function invalidate_inner_signals(fn) {
  var captured = capture_signals(() => untrack(fn));
  for (var signal of captured) {
    if ((signal.f & LEGACY_DERIVED_PROP) !== 0) {
      for (
        const dep of
        /** @type {Derived} */
        signal.deps || []
      ) {
        if ((dep.f & DERIVED) === 0) {
          internal_set(dep, dep.v);
        }
      }
    } else {
      internal_set(signal, signal.v);
    }
  }
}
function untrack(fn) {
  const previous_reaction = active_reaction;
  try {
    active_reaction = null;
    return fn();
  } finally {
    active_reaction = previous_reaction;
  }
}
var STATUS_MASK = ~(DIRTY | MAYBE_DIRTY | CLEAN);
function set_signal_status(signal, status) {
  signal.f = signal.f & STATUS_MASK | status;
}
function push(props, runes = false, fn) {
  component_context = {
    p: component_context,
    c: null,
    e: null,
    m: false,
    s: props,
    x: null,
    l: null
  };
  if (legacy_mode_flag && !runes) {
    component_context.l = {
      s: null,
      u: null,
      r1: [],
      r2: source(false)
    };
  }
  if (dev_fallback_default) {
    component_context.function = fn;
    dev_current_component_function = fn;
  }
}
function pop(component2) {
  const context_stack_item = component_context;
  if (context_stack_item !== null) {
    if (component2 !== void 0) {
      context_stack_item.x = component2;
    }
    const component_effects = context_stack_item.e;
    if (component_effects !== null) {
      var previous_effect = active_effect;
      var previous_reaction = active_reaction;
      context_stack_item.e = null;
      try {
        for (var i = 0; i < component_effects.length; i++) {
          var component_effect = component_effects[i];
          set_active_effect(component_effect.effect);
          set_active_reaction(component_effect.reaction);
          effect(component_effect.fn);
        }
      } finally {
        set_active_effect(previous_effect);
        set_active_reaction(previous_reaction);
      }
    }
    component_context = context_stack_item.p;
    if (dev_fallback_default) {
      dev_current_component_function = context_stack_item.p?.function ?? null;
    }
    context_stack_item.m = true;
  }
  return component2 || /** @type {T} */
  {};
}
function deep_read_state(value) {
  if (typeof value !== "object" || !value || value instanceof EventTarget) {
    return;
  }
  if (STATE_SYMBOL in value) {
    deep_read(value);
  } else if (!Array.isArray(value)) {
    for (let key in value) {
      const prop2 = value[key];
      if (typeof prop2 === "object" && prop2 && STATE_SYMBOL in prop2) {
        deep_read(prop2);
      }
    }
  }
}
function deep_read(value, visited = /* @__PURE__ */ new Set()) {
  if (typeof value === "object" && value !== null && // We don't want to traverse DOM elements
  !(value instanceof EventTarget) && !visited.has(value)) {
    visited.add(value);
    if (value instanceof Date) {
      value.getTime();
    }
    for (let key in value) {
      try {
        deep_read(value[key], visited);
      } catch (e) {
      }
    }
    const proto = get_prototype_of(value);
    if (proto !== Object.prototype && proto !== Array.prototype && proto !== Map.prototype && proto !== Set.prototype && proto !== Date.prototype) {
      const descriptors = get_descriptors(proto);
      for (let key in descriptors) {
        const get3 = descriptors[key].get;
        if (get3) {
          try {
            get3.call(value);
          } catch (e) {
          }
        }
      }
    }
  }
}
if (dev_fallback_default) {
  let throw_rune_error = function(rune) {
    if (!(rune in globalThis)) {
      let value;
      Object.defineProperty(globalThis, rune, {
        configurable: true,
        // eslint-disable-next-line getter-return
        get: () => {
          if (value !== void 0) {
            return value;
          }
          rune_outside_svelte(rune);
        },
        set: (v) => {
          value = v;
        }
      });
    }
  };
  throw_rune_error("$state");
  throw_rune_error("$effect");
  throw_rune_error("$derived");
  throw_rune_error("$inspect");
  throw_rune_error("$props");
  throw_rune_error("$bindable");
}

// node_modules/svelte/src/internal/client/dom/elements/misc.js
function remove_textarea_child(dom) {
  if (hydrating && get_first_child(dom) !== null) {
    clear_text_content(dom);
  }
}
var listening_to_form_reset = false;
function add_form_reset_listener() {
  if (!listening_to_form_reset) {
    listening_to_form_reset = true;
    document.addEventListener(
      "reset",
      (evt) => {
        Promise.resolve().then(() => {
          if (!evt.defaultPrevented) {
            for (
              const e of
              /**@type {HTMLFormElement} */
              evt.target.elements
            ) {
              e.__on_r?.();
            }
          }
        });
      },
      // In the capture phase to guarantee we get noticed of it (no possiblity of stopPropagation)
      { capture: true }
    );
  }
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/shared.js
function without_reactive_context(fn) {
  var previous_reaction = active_reaction;
  var previous_effect = active_effect;
  set_active_reaction(null);
  set_active_effect(null);
  try {
    return fn();
  } finally {
    set_active_reaction(previous_reaction);
    set_active_effect(previous_effect);
  }
}
function listen_to_event_and_reset_event(element2, event2, handler, on_reset = handler) {
  element2.addEventListener(event2, () => without_reactive_context(handler));
  const prev = element2.__on_r;
  if (prev) {
    element2.__on_r = () => {
      prev();
      on_reset(true);
    };
  } else {
    element2.__on_r = () => on_reset(true);
  }
  add_form_reset_listener();
}

// node_modules/svelte/src/internal/client/dom/elements/events.js
var all_registered_events = /* @__PURE__ */ new Set();
var root_event_handles = /* @__PURE__ */ new Set();
function create_event(event_name, dom, handler, options) {
  function target_handler(event2) {
    if (!options.capture) {
      handle_event_propagation.call(dom, event2);
    }
    if (!event2.cancelBubble) {
      return without_reactive_context(() => {
        return handler.call(this, event2);
      });
    }
  }
  if (event_name.startsWith("pointer") || event_name.startsWith("touch") || event_name === "wheel") {
    queue_micro_task(() => {
      dom.addEventListener(event_name, target_handler, options);
    });
  } else {
    dom.addEventListener(event_name, target_handler, options);
  }
  return target_handler;
}
function event(event_name, dom, handler, capture, passive2) {
  var options = { capture, passive: passive2 };
  var target_handler = create_event(event_name, dom, handler, options);
  if (dom === document.body || dom === window || dom === document) {
    teardown(() => {
      dom.removeEventListener(event_name, target_handler, options);
    });
  }
}
function delegate(events) {
  for (var i = 0; i < events.length; i++) {
    all_registered_events.add(events[i]);
  }
  for (var fn of root_event_handles) {
    fn(events);
  }
}
function handle_event_propagation(event2) {
  var handler_element = this;
  var owner_document = (
    /** @type {Node} */
    handler_element.ownerDocument
  );
  var event_name = event2.type;
  var path = event2.composedPath?.() || [];
  var current_target = (
    /** @type {null | Element} */
    path[0] || event2.target
  );
  var path_idx = 0;
  var handled_at = event2.__root;
  if (handled_at) {
    var at_idx = path.indexOf(handled_at);
    if (at_idx !== -1 && (handler_element === document || handler_element === /** @type {any} */
    window)) {
      event2.__root = handler_element;
      return;
    }
    var handler_idx = path.indexOf(handler_element);
    if (handler_idx === -1) {
      return;
    }
    if (at_idx <= handler_idx) {
      path_idx = at_idx;
    }
  }
  current_target = /** @type {Element} */
  path[path_idx] || event2.target;
  if (current_target === handler_element) return;
  define_property(event2, "currentTarget", {
    configurable: true,
    get() {
      return current_target || owner_document;
    }
  });
  var previous_reaction = active_reaction;
  var previous_effect = active_effect;
  set_active_reaction(null);
  set_active_effect(null);
  try {
    var throw_error;
    var other_errors = [];
    while (current_target !== null) {
      var parent_element = current_target.assignedSlot || current_target.parentNode || /** @type {any} */
      current_target.host || null;
      try {
        var delegated = current_target["__" + event_name];
        if (delegated !== void 0 && !/** @type {any} */
        current_target.disabled) {
          if (is_array(delegated)) {
            var [fn, ...data] = delegated;
            fn.apply(current_target, [event2, ...data]);
          } else {
            delegated.call(current_target, event2);
          }
        }
      } catch (error) {
        if (throw_error) {
          other_errors.push(error);
        } else {
          throw_error = error;
        }
      }
      if (event2.cancelBubble || parent_element === handler_element || parent_element === null) {
        break;
      }
      current_target = parent_element;
    }
    if (throw_error) {
      for (let error of other_errors) {
        queueMicrotask(() => {
          throw error;
        });
      }
      throw throw_error;
    }
  } finally {
    event2.__root = handler_element;
    delete event2.currentTarget;
    set_active_reaction(previous_reaction);
    set_active_effect(previous_effect);
  }
}

// node_modules/svelte/src/internal/client/dom/blocks/svelte-head.js
var head_anchor;
function reset_head_anchor() {
  head_anchor = void 0;
}
function head(render_fn) {
  let previous_hydrate_node = null;
  let was_hydrating = hydrating;
  var anchor;
  if (hydrating) {
    previous_hydrate_node = hydrate_node;
    if (head_anchor === void 0) {
      head_anchor = /** @type {TemplateNode} */
      get_first_child(document.head);
    }
    while (head_anchor !== null && (head_anchor.nodeType !== 8 || /** @type {Comment} */
    head_anchor.data !== HYDRATION_START)) {
      head_anchor = /** @type {TemplateNode} */
      get_next_sibling(head_anchor);
    }
    if (head_anchor === null) {
      set_hydrating(false);
    } else {
      head_anchor = set_hydrate_node(
        /** @type {TemplateNode} */
        get_next_sibling(head_anchor)
      );
    }
  }
  if (!hydrating) {
    anchor = document.head.appendChild(create_text());
  }
  try {
    block(() => render_fn(anchor), HEAD_EFFECT);
  } finally {
    if (was_hydrating) {
      set_hydrating(true);
      head_anchor = hydrate_node;
      set_hydrate_node(
        /** @type {TemplateNode} */
        previous_hydrate_node
      );
    }
  }
}

// node_modules/svelte/src/internal/client/dom/reconciler.js
function create_fragment_from_html(html2) {
  var elem = document.createElement("template");
  elem.innerHTML = html2;
  return elem.content;
}

// node_modules/svelte/src/internal/client/dom/template.js
function assign_nodes(start, end) {
  var effect2 = (
    /** @type {Effect} */
    active_effect
  );
  if (effect2.nodes_start === null) {
    effect2.nodes_start = start;
    effect2.nodes_end = end;
  }
}
// @__NO_SIDE_EFFECTS__
function template(content, flags) {
  var is_fragment = (flags & TEMPLATE_FRAGMENT) !== 0;
  var use_import_node = (flags & TEMPLATE_USE_IMPORT_NODE) !== 0;
  var node;
  var has_start = !content.startsWith("<!>");
  return () => {
    if (hydrating) {
      assign_nodes(hydrate_node, null);
      return hydrate_node;
    }
    if (node === void 0) {
      node = create_fragment_from_html(has_start ? content : "<!>" + content);
      if (!is_fragment) node = /** @type {Node} */
      get_first_child(node);
    }
    var clone = (
      /** @type {TemplateNode} */
      use_import_node ? document.importNode(node, true) : node.cloneNode(true)
    );
    if (is_fragment) {
      var start = (
        /** @type {TemplateNode} */
        get_first_child(clone)
      );
      var end = (
        /** @type {TemplateNode} */
        clone.lastChild
      );
      assign_nodes(start, end);
    } else {
      assign_nodes(clone, clone);
    }
    return clone;
  };
}
// @__NO_SIDE_EFFECTS__
function ns_template(content, flags, ns = "svg") {
  var has_start = !content.startsWith("<!>");
  var is_fragment = (flags & TEMPLATE_FRAGMENT) !== 0;
  var wrapped = `<${ns}>${has_start ? content : "<!>" + content}</${ns}>`;
  var node;
  return () => {
    if (hydrating) {
      assign_nodes(hydrate_node, null);
      return hydrate_node;
    }
    if (!node) {
      var fragment = (
        /** @type {DocumentFragment} */
        create_fragment_from_html(wrapped)
      );
      var root13 = (
        /** @type {Element} */
        get_first_child(fragment)
      );
      if (is_fragment) {
        node = document.createDocumentFragment();
        while (get_first_child(root13)) {
          node.appendChild(
            /** @type {Node} */
            get_first_child(root13)
          );
        }
      } else {
        node = /** @type {Element} */
        get_first_child(root13);
      }
    }
    var clone = (
      /** @type {TemplateNode} */
      node.cloneNode(true)
    );
    if (is_fragment) {
      var start = (
        /** @type {TemplateNode} */
        get_first_child(clone)
      );
      var end = (
        /** @type {TemplateNode} */
        clone.lastChild
      );
      assign_nodes(start, end);
    } else {
      assign_nodes(clone, clone);
    }
    return clone;
  };
}
function text(value = "") {
  if (!hydrating) {
    var t = create_text(value + "");
    assign_nodes(t, t);
    return t;
  }
  var node = hydrate_node;
  if (node.nodeType !== 3) {
    node.before(node = create_text());
    set_hydrate_node(node);
  }
  assign_nodes(node, node);
  return node;
}
function comment() {
  if (hydrating) {
    assign_nodes(hydrate_node, null);
    return hydrate_node;
  }
  var frag = document.createDocumentFragment();
  var start = document.createComment("");
  var anchor = create_text();
  frag.append(start, anchor);
  assign_nodes(start, anchor);
  return frag;
}
function append(anchor, dom) {
  if (hydrating) {
    active_effect.nodes_end = hydrate_node;
    hydrate_next();
    return;
  }
  if (anchor === null) {
    return;
  }
  anchor.before(
    /** @type {Node} */
    dom
  );
}

// node_modules/svelte/src/utils.js
var DOM_BOOLEAN_ATTRIBUTES = [
  "allowfullscreen",
  "async",
  "autofocus",
  "autoplay",
  "checked",
  "controls",
  "default",
  "disabled",
  "formnovalidate",
  "hidden",
  "indeterminate",
  "ismap",
  "loop",
  "multiple",
  "muted",
  "nomodule",
  "novalidate",
  "open",
  "playsinline",
  "readonly",
  "required",
  "reversed",
  "seamless",
  "selected",
  "webkitdirectory"
];
var DOM_PROPERTIES = [
  ...DOM_BOOLEAN_ATTRIBUTES,
  "formNoValidate",
  "isMap",
  "noModule",
  "playsInline",
  "readOnly",
  "value",
  "inert",
  "volume",
  "defaultValue",
  "defaultChecked",
  "srcObject"
];
var PASSIVE_EVENTS = ["touchstart", "touchmove"];
function is_passive_event(name) {
  return PASSIVE_EVENTS.includes(name);
}

// node_modules/svelte/src/internal/client/render.js
var should_intro = true;
function set_text(text2, value) {
  var str = value == null ? "" : typeof value === "object" ? value + "" : value;
  if (str !== (text2.__t ??= text2.nodeValue)) {
    text2.__t = str;
    text2.nodeValue = str == null ? "" : str + "";
  }
}
function mount(component2, options) {
  return _mount(component2, options);
}
function hydrate(component2, options) {
  init_operations();
  options.intro = options.intro ?? false;
  const target2 = options.target;
  const was_hydrating = hydrating;
  const previous_hydrate_node = hydrate_node;
  try {
    var anchor = (
      /** @type {TemplateNode} */
      get_first_child(target2)
    );
    while (anchor && (anchor.nodeType !== 8 || /** @type {Comment} */
    anchor.data !== HYDRATION_START)) {
      anchor = /** @type {TemplateNode} */
      get_next_sibling(anchor);
    }
    if (!anchor) {
      throw HYDRATION_ERROR;
    }
    set_hydrating(true);
    set_hydrate_node(
      /** @type {Comment} */
      anchor
    );
    hydrate_next();
    const instance = _mount(component2, { ...options, anchor });
    if (hydrate_node === null || hydrate_node.nodeType !== 8 || /** @type {Comment} */
    hydrate_node.data !== HYDRATION_END) {
      hydration_mismatch();
      throw HYDRATION_ERROR;
    }
    set_hydrating(false);
    return (
      /**  @type {Exports} */
      instance
    );
  } catch (error) {
    if (error === HYDRATION_ERROR) {
      if (options.recover === false) {
        hydration_failed();
      }
      init_operations();
      clear_text_content(target2);
      set_hydrating(false);
      return mount(component2, options);
    }
    throw error;
  } finally {
    set_hydrating(was_hydrating);
    set_hydrate_node(previous_hydrate_node);
    reset_head_anchor();
  }
}
var document_listeners = /* @__PURE__ */ new Map();
function _mount(Component, { target: target2, anchor, props = {}, events, context, intro = true }) {
  init_operations();
  var registered_events = /* @__PURE__ */ new Set();
  var event_handle = (events2) => {
    for (var i = 0; i < events2.length; i++) {
      var event_name = events2[i];
      if (registered_events.has(event_name)) continue;
      registered_events.add(event_name);
      var passive2 = is_passive_event(event_name);
      target2.addEventListener(event_name, handle_event_propagation, { passive: passive2 });
      var n = document_listeners.get(event_name);
      if (n === void 0) {
        document.addEventListener(event_name, handle_event_propagation, { passive: passive2 });
        document_listeners.set(event_name, 1);
      } else {
        document_listeners.set(event_name, n + 1);
      }
    }
  };
  event_handle(array_from(all_registered_events));
  root_event_handles.add(event_handle);
  var component2 = void 0;
  var unmount2 = effect_root(() => {
    var anchor_node = anchor ?? target2.appendChild(create_text());
    branch(() => {
      if (context) {
        push({});
        var ctx = (
          /** @type {ComponentContext} */
          component_context
        );
        ctx.c = context;
      }
      if (events) {
        props.$$events = events;
      }
      if (hydrating) {
        assign_nodes(
          /** @type {TemplateNode} */
          anchor_node,
          null
        );
      }
      should_intro = intro;
      component2 = Component(anchor_node, props) || {};
      should_intro = true;
      if (hydrating) {
        active_effect.nodes_end = hydrate_node;
      }
      if (context) {
        pop();
      }
    });
    return () => {
      for (var event_name of registered_events) {
        target2.removeEventListener(event_name, handle_event_propagation);
        var n = (
          /** @type {number} */
          document_listeners.get(event_name)
        );
        if (--n === 0) {
          document.removeEventListener(event_name, handle_event_propagation);
          document_listeners.delete(event_name);
        } else {
          document_listeners.set(event_name, n);
        }
      }
      root_event_handles.delete(event_handle);
      mounted_components.delete(component2);
      if (anchor_node !== anchor) {
        anchor_node.parentNode?.removeChild(anchor_node);
      }
    };
  });
  mounted_components.set(component2, unmount2);
  return component2;
}
var mounted_components = /* @__PURE__ */ new WeakMap();
function unmount(component2) {
  const fn = mounted_components.get(component2);
  if (fn) {
    fn();
  } else if (dev_fallback_default) {
    lifecycle_double_unmount();
  }
}

// node_modules/svelte/src/internal/client/dom/blocks/if.js
function if_block(node, fn, elseif = false) {
  if (hydrating) {
    hydrate_next();
  }
  var anchor = node;
  var consequent_effect = null;
  var alternate_effect = null;
  var condition = UNINITIALIZED;
  var flags = elseif ? EFFECT_TRANSPARENT : 0;
  var has_branch = false;
  const set_branch = (fn2, flag = true) => {
    has_branch = true;
    update_branch(flag, fn2);
  };
  const update_branch = (new_condition, fn2) => {
    if (condition === (condition = new_condition)) return;
    let mismatch = false;
    if (hydrating) {
      const is_else = (
        /** @type {Comment} */
        anchor.data === HYDRATION_START_ELSE
      );
      if (!!condition === is_else) {
        anchor = remove_nodes();
        set_hydrate_node(anchor);
        set_hydrating(false);
        mismatch = true;
      }
    }
    if (condition) {
      if (consequent_effect) {
        resume_effect(consequent_effect);
      } else if (fn2) {
        consequent_effect = branch(() => fn2(anchor));
      }
      if (alternate_effect) {
        pause_effect(alternate_effect, () => {
          alternate_effect = null;
        });
      }
    } else {
      if (alternate_effect) {
        resume_effect(alternate_effect);
      } else if (fn2) {
        alternate_effect = branch(() => fn2(anchor));
      }
      if (consequent_effect) {
        pause_effect(consequent_effect, () => {
          consequent_effect = null;
        });
      }
    }
    if (mismatch) {
      set_hydrating(true);
    }
  };
  block(() => {
    has_branch = false;
    fn(set_branch);
    if (!has_branch) {
      update_branch(null, null);
    }
  }, flags);
  if (hydrating) {
    anchor = hydrate_node;
  }
}

// node_modules/svelte/src/internal/client/dom/blocks/key.js
function key_block(node, get_key, render_fn) {
  if (hydrating) {
    hydrate_next();
  }
  var anchor = node;
  var key = UNINITIALIZED;
  var effect2;
  var changed = is_runes() ? not_equal : safe_not_equal;
  block(() => {
    if (changed(key, key = get_key())) {
      if (effect2) {
        pause_effect(effect2);
      }
      effect2 = branch(() => render_fn(anchor));
    }
  });
  if (hydrating) {
    anchor = hydrate_node;
  }
}

// node_modules/svelte/src/internal/client/dom/blocks/each.js
var current_each_item = null;
function index(_, i) {
  return i;
}
function pause_effects(state2, items, controlled_anchor, items_map) {
  var transitions = [];
  var length = items.length;
  for (var i = 0; i < length; i++) {
    pause_children(items[i].e, transitions, true);
  }
  var is_controlled = length > 0 && transitions.length === 0 && controlled_anchor !== null;
  if (is_controlled) {
    var parent_node = (
      /** @type {Element} */
      /** @type {Element} */
      controlled_anchor.parentNode
    );
    clear_text_content(parent_node);
    parent_node.append(
      /** @type {Element} */
      controlled_anchor
    );
    items_map.clear();
    link(state2, items[0].prev, items[length - 1].next);
  }
  run_out_transitions(transitions, () => {
    for (var i2 = 0; i2 < length; i2++) {
      var item = items[i2];
      if (!is_controlled) {
        items_map.delete(item.k);
        link(state2, item.prev, item.next);
      }
      destroy_effect(item.e, !is_controlled);
    }
  });
}
function each(node, flags, get_collection, get_key, render_fn, fallback_fn = null) {
  var anchor = node;
  var state2 = { flags, items: /* @__PURE__ */ new Map(), first: null };
  var is_controlled = (flags & EACH_IS_CONTROLLED) !== 0;
  if (is_controlled) {
    var parent_node = (
      /** @type {Element} */
      node
    );
    anchor = hydrating ? set_hydrate_node(
      /** @type {Comment | Text} */
      get_first_child(parent_node)
    ) : parent_node.appendChild(create_text());
  }
  if (hydrating) {
    hydrate_next();
  }
  var fallback2 = null;
  var was_empty = false;
  block(() => {
    var collection = get_collection();
    var array = is_array(collection) ? collection : collection == null ? [] : array_from(collection);
    var length = array.length;
    if (was_empty && length === 0) {
      return;
    }
    was_empty = length === 0;
    let mismatch = false;
    if (hydrating) {
      var is_else = (
        /** @type {Comment} */
        anchor.data === HYDRATION_START_ELSE
      );
      if (is_else !== (length === 0)) {
        anchor = remove_nodes();
        set_hydrate_node(anchor);
        set_hydrating(false);
        mismatch = true;
      }
    }
    if (hydrating) {
      var prev = null;
      var item;
      for (var i = 0; i < length; i++) {
        if (hydrate_node.nodeType === 8 && /** @type {Comment} */
        hydrate_node.data === HYDRATION_END) {
          anchor = /** @type {Comment} */
          hydrate_node;
          mismatch = true;
          set_hydrating(false);
          break;
        }
        var value = array[i];
        var key = get_key(value, i);
        item = create_item(hydrate_node, state2, prev, null, value, key, i, render_fn, flags);
        state2.items.set(key, item);
        prev = item;
      }
      if (length > 0) {
        set_hydrate_node(remove_nodes());
      }
    }
    if (!hydrating) {
      var effect2 = (
        /** @type {Effect} */
        active_reaction
      );
      reconcile(array, state2, anchor, render_fn, flags, (effect2.f & INERT) !== 0, get_key);
    }
    if (fallback_fn !== null) {
      if (length === 0) {
        if (fallback2) {
          resume_effect(fallback2);
        } else {
          fallback2 = branch(() => fallback_fn(anchor));
        }
      } else if (fallback2 !== null) {
        pause_effect(fallback2, () => {
          fallback2 = null;
        });
      }
    }
    if (mismatch) {
      set_hydrating(true);
    }
    get_collection();
  });
  if (hydrating) {
    anchor = hydrate_node;
  }
}
function reconcile(array, state2, anchor, render_fn, flags, is_inert, get_key) {
  var is_animated = (flags & EACH_IS_ANIMATED) !== 0;
  var should_update = (flags & (EACH_ITEM_REACTIVE | EACH_INDEX_REACTIVE)) !== 0;
  var length = array.length;
  var items = state2.items;
  var first = state2.first;
  var current = first;
  var seen;
  var prev = null;
  var to_animate;
  var matched = [];
  var stashed = [];
  var value;
  var key;
  var item;
  var i;
  if (is_animated) {
    for (i = 0; i < length; i += 1) {
      value = array[i];
      key = get_key(value, i);
      item = items.get(key);
      if (item !== void 0) {
        item.a?.measure();
        (to_animate ??= /* @__PURE__ */ new Set()).add(item);
      }
    }
  }
  for (i = 0; i < length; i += 1) {
    value = array[i];
    key = get_key(value, i);
    item = items.get(key);
    if (item === void 0) {
      var child_anchor = current ? (
        /** @type {TemplateNode} */
        current.e.nodes_start
      ) : anchor;
      prev = create_item(
        child_anchor,
        state2,
        prev,
        prev === null ? state2.first : prev.next,
        value,
        key,
        i,
        render_fn,
        flags
      );
      items.set(key, prev);
      matched = [];
      stashed = [];
      current = prev.next;
      continue;
    }
    if (should_update) {
      update_item(item, value, i, flags);
    }
    if ((item.e.f & INERT) !== 0) {
      resume_effect(item.e);
      if (is_animated) {
        item.a?.unfix();
        (to_animate ??= /* @__PURE__ */ new Set()).delete(item);
      }
    }
    if (item !== current) {
      if (seen !== void 0 && seen.has(item)) {
        if (matched.length < stashed.length) {
          var start = stashed[0];
          var j;
          prev = start.prev;
          var a = matched[0];
          var b = matched[matched.length - 1];
          for (j = 0; j < matched.length; j += 1) {
            move(matched[j], start, anchor);
          }
          for (j = 0; j < stashed.length; j += 1) {
            seen.delete(stashed[j]);
          }
          link(state2, a.prev, b.next);
          link(state2, prev, a);
          link(state2, b, start);
          current = start;
          prev = b;
          i -= 1;
          matched = [];
          stashed = [];
        } else {
          seen.delete(item);
          move(item, current, anchor);
          link(state2, item.prev, item.next);
          link(state2, item, prev === null ? state2.first : prev.next);
          link(state2, prev, item);
          prev = item;
        }
        continue;
      }
      matched = [];
      stashed = [];
      while (current !== null && current.k !== key) {
        if (is_inert || (current.e.f & INERT) === 0) {
          (seen ??= /* @__PURE__ */ new Set()).add(current);
        }
        stashed.push(current);
        current = current.next;
      }
      if (current === null) {
        continue;
      }
      item = current;
    }
    matched.push(item);
    prev = item;
    current = item.next;
  }
  if (current !== null || seen !== void 0) {
    var to_destroy = seen === void 0 ? [] : array_from(seen);
    while (current !== null) {
      if (is_inert || (current.e.f & INERT) === 0) {
        to_destroy.push(current);
      }
      current = current.next;
    }
    var destroy_length = to_destroy.length;
    if (destroy_length > 0) {
      var controlled_anchor = (flags & EACH_IS_CONTROLLED) !== 0 && length === 0 ? anchor : null;
      if (is_animated) {
        for (i = 0; i < destroy_length; i += 1) {
          to_destroy[i].a?.measure();
        }
        for (i = 0; i < destroy_length; i += 1) {
          to_destroy[i].a?.fix();
        }
      }
      pause_effects(state2, to_destroy, controlled_anchor, items);
    }
  }
  if (is_animated) {
    queue_micro_task(() => {
      if (to_animate === void 0) return;
      for (item of to_animate) {
        item.a?.apply();
      }
    });
  }
  active_effect.first = state2.first && state2.first.e;
  active_effect.last = prev && prev.e;
}
function update_item(item, value, index2, type) {
  if ((type & EACH_ITEM_REACTIVE) !== 0) {
    internal_set(item.v, value);
  }
  if ((type & EACH_INDEX_REACTIVE) !== 0) {
    internal_set(
      /** @type {Value<number>} */
      item.i,
      index2
    );
  } else {
    item.i = index2;
  }
}
function create_item(anchor, state2, prev, next2, value, key, index2, render_fn, flags) {
  var previous_each_item = current_each_item;
  var reactive = (flags & EACH_ITEM_REACTIVE) !== 0;
  var mutable = (flags & EACH_ITEM_IMMUTABLE) === 0;
  var v = reactive ? mutable ? mutable_source(value) : source(value) : value;
  var i = (flags & EACH_INDEX_REACTIVE) === 0 ? index2 : source(index2);
  var item = {
    i,
    v,
    k: key,
    a: null,
    // @ts-expect-error
    e: null,
    prev,
    next: next2
  };
  current_each_item = item;
  try {
    item.e = branch(() => render_fn(anchor, v, i), hydrating);
    item.e.prev = prev && prev.e;
    item.e.next = next2 && next2.e;
    if (prev === null) {
      state2.first = item;
    } else {
      prev.next = item;
      prev.e.next = item.e;
    }
    if (next2 !== null) {
      next2.prev = item;
      next2.e.prev = item.e;
    }
    return item;
  } finally {
    current_each_item = previous_each_item;
  }
}
function move(item, next2, anchor) {
  var end = item.next ? (
    /** @type {TemplateNode} */
    item.next.e.nodes_start
  ) : anchor;
  var dest = next2 ? (
    /** @type {TemplateNode} */
    next2.e.nodes_start
  ) : anchor;
  var node = (
    /** @type {TemplateNode} */
    item.e.nodes_start
  );
  while (node !== end) {
    var next_node = (
      /** @type {TemplateNode} */
      get_next_sibling(node)
    );
    dest.before(node);
    node = next_node;
  }
}
function link(state2, prev, next2) {
  if (prev === null) {
    state2.first = next2;
  } else {
    prev.next = next2;
    prev.e.next = next2 && next2.e;
  }
  if (next2 !== null) {
    next2.prev = prev;
    next2.e.prev = prev && prev.e;
  }
}

// node_modules/svelte/src/internal/client/dom/blocks/slot.js
function slot(anchor, $$props, name, slot_props, fallback_fn) {
  if (hydrating) {
    hydrate_next();
  }
  var slot_fn = $$props.$$slots?.[name];
  var is_interop = false;
  if (slot_fn === true) {
    slot_fn = $$props[name === "default" ? "children" : name];
    is_interop = true;
  }
  if (slot_fn === void 0) {
    if (fallback_fn !== null) {
      fallback_fn(anchor);
    }
  } else {
    slot_fn(anchor, is_interop ? () => slot_props : slot_props);
  }
}

// node_modules/svelte/src/internal/client/dom/elements/actions.js
function action(dom, action2, get_value) {
  effect(() => {
    var payload = untrack(() => action2(dom, get_value?.()) || {});
    if (get_value && payload?.update) {
      var inited = false;
      var prev = (
        /** @type {any} */
        {}
      );
      render_effect(() => {
        var value = get_value();
        deep_read_state(value);
        if (inited && safe_not_equal(prev, value)) {
          prev = value;
          payload.update(value);
        }
      });
      inited = true;
    }
    if (payload?.destroy) {
      return () => (
        /** @type {Function} */
        payload.destroy()
      );
    }
  });
}

// node_modules/svelte/src/internal/client/dom/elements/attributes.js
function remove_input_defaults(input) {
  if (!hydrating) return;
  var already_removed = false;
  var remove_defaults = () => {
    if (already_removed) return;
    already_removed = true;
    if (input.hasAttribute("value")) {
      var value = input.value;
      set_attribute(input, "value", null);
      input.value = value;
    }
    if (input.hasAttribute("checked")) {
      var checked = input.checked;
      set_attribute(input, "checked", null);
      input.checked = checked;
    }
  };
  input.__on_r = remove_defaults;
  queue_idle_task(remove_defaults);
  add_form_reset_listener();
}
function set_value(element2, value) {
  var attributes = element2.__attributes ??= {};
  if (attributes.value === (attributes.value = // treat null and undefined the same for the initial value
  value ?? void 0) || // @ts-expect-error
  // `progress` elements always need their value set when its `0`
  element2.value === value && (value !== 0 || element2.nodeName !== "PROGRESS")) {
    return;
  }
  element2.value = value;
}
function set_checked(element2, checked) {
  var attributes = element2.__attributes ??= {};
  if (attributes.checked === (attributes.checked = // treat null and undefined the same for the initial value
  checked ?? void 0)) {
    return;
  }
  element2.checked = checked;
}
function set_attribute(element2, attribute, value, skip_warning) {
  var attributes = element2.__attributes ??= {};
  if (hydrating) {
    attributes[attribute] = element2.getAttribute(attribute);
    if (attribute === "src" || attribute === "srcset" || attribute === "href" && element2.nodeName === "LINK") {
      if (!skip_warning) {
        check_src_in_dev_hydration(element2, attribute, value ?? "");
      }
      return;
    }
  }
  if (attributes[attribute] === (attributes[attribute] = value)) return;
  if (attribute === "style" && "__styles" in element2) {
    element2.__styles = {};
  }
  if (attribute === "loading") {
    element2[LOADING_ATTR_SYMBOL] = value;
  }
  if (value == null) {
    element2.removeAttribute(attribute);
  } else if (typeof value !== "string" && get_setters(element2).includes(attribute)) {
    element2[attribute] = value;
  } else {
    element2.setAttribute(attribute, value);
  }
}
var setters_cache = /* @__PURE__ */ new Map();
function get_setters(element2) {
  var setters = setters_cache.get(element2.nodeName);
  if (setters) return setters;
  setters_cache.set(element2.nodeName, setters = []);
  var descriptors;
  var proto = element2;
  var element_proto = Element.prototype;
  while (element_proto !== proto) {
    descriptors = get_descriptors(proto);
    for (var key in descriptors) {
      if (descriptors[key].set) {
        setters.push(key);
      }
    }
    proto = get_prototype_of(proto);
  }
  return setters;
}
function check_src_in_dev_hydration(element2, attribute, value) {
  if (!dev_fallback_default) return;
  if (attribute === "srcset" && srcset_url_equal(element2, value)) return;
  if (src_url_equal(element2.getAttribute(attribute) ?? "", value)) return;
  hydration_attribute_changed(
    attribute,
    element2.outerHTML.replace(element2.innerHTML, element2.innerHTML && "..."),
    String(value)
  );
}
function src_url_equal(element_src, url) {
  if (element_src === url) return true;
  return new URL(element_src, document.baseURI).href === new URL(url, document.baseURI).href;
}
function split_srcset(srcset) {
  return srcset.split(",").map((src) => src.trim().split(" ").filter(Boolean));
}
function srcset_url_equal(element2, srcset) {
  var element_urls = split_srcset(element2.srcset);
  var urls = split_srcset(srcset);
  return urls.length === element_urls.length && urls.every(
    ([url, width], i) => width === element_urls[i][1] && // We need to test both ways because Vite will create an a full URL with
    // `new URL(asset, import.meta.url).href` for the client when `base: './'`, and the
    // relative URLs inside srcset are not automatically resolved to absolute URLs by
    // browsers (in contrast to img.src). This means both SSR and DOM code could
    // contain relative or absolute URLs.
    (src_url_equal(element_urls[i][0], url) || src_url_equal(url, element_urls[i][0]))
  );
}

// node_modules/svelte/src/internal/client/dom/elements/class.js
function set_class(dom, value) {
  var prev_class_name = dom.__className;
  var next_class_name = to_class(value);
  if (hydrating && dom.className === next_class_name) {
    dom.__className = next_class_name;
  } else if (prev_class_name !== next_class_name || hydrating && dom.className !== next_class_name) {
    if (value == null) {
      dom.removeAttribute("class");
    } else {
      dom.className = next_class_name;
    }
    dom.__className = next_class_name;
  }
}
function to_class(value) {
  return value == null ? "" : value;
}
function toggle_class(dom, class_name, value) {
  if (value) {
    if (dom.classList.contains(class_name)) return;
    dom.classList.add(class_name);
  } else {
    if (!dom.classList.contains(class_name)) return;
    dom.classList.remove(class_name);
  }
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/input.js
function bind_value(input, get3, set2 = get3) {
  var runes = is_runes();
  listen_to_event_and_reset_event(input, "input", (is_reset) => {
    if (dev_fallback_default && input.type === "checkbox") {
      bind_invalid_checkbox_value();
    }
    var value = is_reset ? input.defaultValue : input.value;
    value = is_numberlike_input(input) ? to_number(value) : value;
    set2(value);
    if (runes && value !== (value = get3())) {
      var start = input.selectionStart;
      var end = input.selectionEnd;
      input.value = value ?? "";
      if (end !== null) {
        input.selectionStart = start;
        input.selectionEnd = Math.min(end, input.value.length);
      }
    }
  });
  if (
    // If we are hydrating and the value has since changed,
    // then use the updated value from the input instead.
    hydrating && input.defaultValue !== input.value || // If defaultValue is set, then value == defaultValue
    // TODO Svelte 6: remove input.value check and set to empty string?
    untrack(get3) == null && input.value
  ) {
    set2(is_numberlike_input(input) ? to_number(input.value) : input.value);
  }
  render_effect(() => {
    if (dev_fallback_default && input.type === "checkbox") {
      bind_invalid_checkbox_value();
    }
    var value = get3();
    if (is_numberlike_input(input) && value === to_number(input.value)) {
      return;
    }
    if (input.type === "date" && !value && !input.value) {
      return;
    }
    if (value !== input.value) {
      input.value = value ?? "";
    }
  });
}
function is_numberlike_input(input) {
  var type = input.type;
  return type === "number" || type === "range";
}
function to_number(value) {
  return value === "" ? null : +value;
}

// node_modules/svelte/src/internal/client/dom/elements/bindings/select.js
function select_option(select, value, mounting) {
  if (select.multiple) {
    return select_options(select, value);
  }
  for (var option of select.options) {
    var option_value = get_option_value(option);
    if (is(option_value, value)) {
      option.selected = true;
      return;
    }
  }
  if (!mounting || value !== void 0) {
    select.selectedIndex = -1;
  }
}
function init_select(select, get_value) {
  let mounting = true;
  effect(() => {
    if (get_value) {
      select_option(select, untrack(get_value), mounting);
    }
    mounting = false;
    var observer = new MutationObserver(() => {
      var value = select.__value;
      select_option(select, value);
    });
    observer.observe(select, {
      // Listen to option element changes
      childList: true,
      subtree: true,
      // because of <optgroup>
      // Listen to option element value attribute changes
      // (doesn't get notified of select value changes,
      // because that property is not reflected as an attribute)
      attributes: true,
      attributeFilter: ["value"]
    });
    return () => {
      observer.disconnect();
    };
  });
}
function bind_select_value(select, get3, set2 = get3) {
  var mounting = true;
  listen_to_event_and_reset_event(select, "change", (is_reset) => {
    var query = is_reset ? "[selected]" : ":checked";
    var value;
    if (select.multiple) {
      value = [].map.call(select.querySelectorAll(query), get_option_value);
    } else {
      var selected_option = select.querySelector(query) ?? // will fall back to first non-disabled option if no option is selected
      select.querySelector("option:not([disabled])");
      value = selected_option && get_option_value(selected_option);
    }
    set2(value);
  });
  effect(() => {
    var value = get3();
    select_option(select, value, mounting);
    if (mounting && value === void 0) {
      var selected_option = select.querySelector(":checked");
      if (selected_option !== null) {
        value = get_option_value(selected_option);
        set2(value);
      }
    }
    select.__value = value;
    mounting = false;
  });
  init_select(select);
}
function select_options(select, value) {
  for (var option of select.options) {
    option.selected = ~value.indexOf(get_option_value(option));
  }
}
function get_option_value(option) {
  if ("__value" in option) {
    return option.__value;
  } else {
    return option.value;
  }
}

// node_modules/svelte/src/internal/client/dom/legacy/event-modifiers.js
function preventDefault(fn) {
  return function(...args) {
    var event2 = (
      /** @type {Event} */
      args[0]
    );
    event2.preventDefault();
    return fn?.apply(this, args);
  };
}

// node_modules/svelte/src/internal/client/dom/legacy/lifecycle.js
function init(immutable = false) {
  const context = (
    /** @type {ComponentContextLegacy} */
    component_context
  );
  const callbacks = context.l.u;
  if (!callbacks) return;
  let props = () => deep_read_state(context.s);
  if (immutable) {
    let version = 0;
    let prev = (
      /** @type {Record<string, any>} */
      {}
    );
    const d = derived(() => {
      let changed = false;
      const props2 = context.s;
      for (const key in props2) {
        if (props2[key] !== prev[key]) {
          prev[key] = props2[key];
          changed = true;
        }
      }
      if (changed) version++;
      return version;
    });
    props = () => get(d);
  }
  if (callbacks.b.length) {
    user_pre_effect(() => {
      observe_all(context, props);
      run_all(callbacks.b);
    });
  }
  user_effect(() => {
    const fns = untrack(() => callbacks.m.map(run));
    return () => {
      for (const fn of fns) {
        if (typeof fn === "function") {
          fn();
        }
      }
    };
  });
  if (callbacks.a.length) {
    user_effect(() => {
      observe_all(context, props);
      run_all(callbacks.a);
    });
  }
}
function observe_all(context, props) {
  if (context.l.s) {
    for (const signal of context.l.s) get(signal);
  }
  props();
}

// node_modules/svelte/src/internal/client/dom/legacy/misc.js
function bubble_event($$props, event2) {
  var events = (
    /** @type {Record<string, Function[] | Function>} */
    $$props.$$events?.[event2.type]
  );
  var callbacks = is_array(events) ? events.slice() : events == null ? [] : [events];
  for (var fn of callbacks) {
    fn.call(this, event2);
  }
}

// node_modules/svelte/src/store/utils.js
function subscribe_to_store(store, run2, invalidate) {
  if (store == null) {
    run2(void 0);
    if (invalidate) invalidate(void 0);
    return noop;
  }
  const unsub = untrack(
    () => store.subscribe(
      run2,
      // @ts-expect-error
      invalidate
    )
  );
  return unsub.unsubscribe ? () => unsub.unsubscribe() : unsub;
}

// node_modules/svelte/src/internal/client/reactivity/store.js
var is_store_binding = false;
function store_get(store, store_name, stores) {
  const entry = stores[store_name] ??= {
    store: null,
    source: mutable_source(void 0),
    unsubscribe: noop
  };
  if (entry.store !== store) {
    entry.unsubscribe();
    entry.store = store ?? null;
    if (store == null) {
      entry.source.v = void 0;
      entry.unsubscribe = noop;
    } else {
      var is_synchronous_callback = true;
      entry.unsubscribe = subscribe_to_store(store, (v) => {
        if (is_synchronous_callback) {
          entry.source.v = v;
        } else {
          set(entry.source, v);
        }
      });
      is_synchronous_callback = false;
    }
  }
  return get(entry.source);
}
function setup_stores() {
  const stores = {};
  teardown(() => {
    for (var store_name in stores) {
      const ref = stores[store_name];
      ref.unsubscribe();
    }
  });
  return stores;
}
function capture_store_binding(fn) {
  var previous_is_store_binding = is_store_binding;
  try {
    is_store_binding = false;
    return [fn(), is_store_binding];
  } finally {
    is_store_binding = previous_is_store_binding;
  }
}

// node_modules/svelte/src/internal/client/reactivity/props.js
function with_parent_branch(fn) {
  var effect2 = active_effect;
  var previous_effect = active_effect;
  while (effect2 !== null && (effect2.f & (BRANCH_EFFECT | ROOT_EFFECT)) === 0) {
    effect2 = effect2.parent;
  }
  try {
    set_active_effect(effect2);
    return fn();
  } finally {
    set_active_effect(previous_effect);
  }
}
function prop(props, key, flags, fallback2) {
  var immutable = (flags & PROPS_IS_IMMUTABLE) !== 0;
  var runes = !legacy_mode_flag || (flags & PROPS_IS_RUNES) !== 0;
  var bindable = (flags & PROPS_IS_BINDABLE) !== 0;
  var lazy = (flags & PROPS_IS_LAZY_INITIAL) !== 0;
  var is_store_sub = false;
  var prop_value;
  if (bindable) {
    [prop_value, is_store_sub] = capture_store_binding(() => (
      /** @type {V} */
      props[key]
    ));
  } else {
    prop_value = /** @type {V} */
    props[key];
  }
  var is_entry_props = STATE_SYMBOL in props || LEGACY_PROPS in props;
  var setter = get_descriptor(props, key)?.set ?? (is_entry_props && bindable && key in props ? (v) => props[key] = v : void 0);
  var fallback_value = (
    /** @type {V} */
    fallback2
  );
  var fallback_dirty = true;
  var fallback_used = false;
  var get_fallback = () => {
    fallback_used = true;
    if (fallback_dirty) {
      fallback_dirty = false;
      if (lazy) {
        fallback_value = untrack(
          /** @type {() => V} */
          fallback2
        );
      } else {
        fallback_value = /** @type {V} */
        fallback2;
      }
    }
    return fallback_value;
  };
  if (prop_value === void 0 && fallback2 !== void 0) {
    if (setter && runes) {
      props_invalid_value(key);
    }
    prop_value = get_fallback();
    if (setter) setter(prop_value);
  }
  var getter;
  if (runes) {
    getter = () => {
      var value = (
        /** @type {V} */
        props[key]
      );
      if (value === void 0) return get_fallback();
      fallback_dirty = true;
      fallback_used = false;
      return value;
    };
  } else {
    var derived_getter = with_parent_branch(
      () => (immutable ? derived : derived_safe_equal)(() => (
        /** @type {V} */
        props[key]
      ))
    );
    derived_getter.f |= LEGACY_DERIVED_PROP;
    getter = () => {
      var value = get(derived_getter);
      if (value !== void 0) fallback_value = /** @type {V} */
      void 0;
      return value === void 0 ? fallback_value : value;
    };
  }
  if ((flags & PROPS_IS_UPDATED) === 0) {
    return getter;
  }
  if (setter) {
    var legacy_parent = props.$$legacy;
    return function(value, mutation) {
      if (arguments.length > 0) {
        if (!runes || !mutation || legacy_parent || is_store_sub) {
          setter(mutation ? getter() : value);
        }
        return value;
      } else {
        return getter();
      }
    };
  }
  var from_child = false;
  var was_from_child = false;
  var inner_current_value = mutable_source(prop_value);
  var current_value = with_parent_branch(
    () => derived(() => {
      var parent_value = getter();
      var child_value = get(inner_current_value);
      if (from_child) {
        from_child = false;
        was_from_child = true;
        return child_value;
      }
      was_from_child = false;
      return inner_current_value.v = parent_value;
    })
  );
  if (!immutable) current_value.equals = safe_equals;
  return function(value, mutation) {
    if (captured_signals !== null) {
      from_child = was_from_child;
      getter();
      get(inner_current_value);
    }
    if (arguments.length > 0) {
      const new_value = mutation ? get(current_value) : runes && bindable ? proxy(value) : value;
      if (!current_value.equals(new_value)) {
        from_child = true;
        set(inner_current_value, new_value);
        if (fallback_used && fallback_value !== void 0) {
          fallback_value = new_value;
        }
        untrack(() => get(current_value));
      }
      return value;
    }
    return get(current_value);
  };
}

// node_modules/svelte/src/legacy/legacy-client.js
function createClassComponent(options) {
  return new Svelte4Component(options);
}
var Svelte4Component = class {
  /** @type {any} */
  #events;
  /** @type {Record<string, any>} */
  #instance;
  /**
   * @param {ComponentConstructorOptions & {
   *  component: any;
   * }} options
   */
  constructor(options) {
    var sources = /* @__PURE__ */ new Map();
    var add_source = (key, value) => {
      var s = mutable_source(value);
      sources.set(key, s);
      return s;
    };
    const props = new Proxy(
      { ...options.props || {}, $$events: {} },
      {
        get(target2, prop2) {
          return get(sources.get(prop2) ?? add_source(prop2, Reflect.get(target2, prop2)));
        },
        has(target2, prop2) {
          if (prop2 === LEGACY_PROPS) return true;
          get(sources.get(prop2) ?? add_source(prop2, Reflect.get(target2, prop2)));
          return Reflect.has(target2, prop2);
        },
        set(target2, prop2, value) {
          set(sources.get(prop2) ?? add_source(prop2, value), value);
          return Reflect.set(target2, prop2, value);
        }
      }
    );
    this.#instance = (options.hydrate ? hydrate : mount)(options.component, {
      target: options.target,
      anchor: options.anchor,
      props,
      context: options.context,
      intro: options.intro ?? false,
      recover: options.recover
    });
    if (!options?.props?.$$host || options.sync === false) {
      flush_sync();
    }
    this.#events = props.$$events;
    for (const key of Object.keys(this.#instance)) {
      if (key === "$set" || key === "$destroy" || key === "$on") continue;
      define_property(this, key, {
        get() {
          return this.#instance[key];
        },
        /** @param {any} value */
        set(value) {
          this.#instance[key] = value;
        },
        enumerable: true
      });
    }
    this.#instance.$set = /** @param {Record<string, any>} next */
    (next2) => {
      Object.assign(props, next2);
    };
    this.#instance.$destroy = () => {
      unmount(this.#instance);
    };
  }
  /** @param {Record<string, any>} props */
  $set(props) {
    this.#instance.$set(props);
  }
  /**
   * @param {string} event
   * @param {(...args: any[]) => any} callback
   * @returns {any}
   */
  $on(event2, callback) {
    this.#events[event2] = this.#events[event2] || [];
    const cb = (...args) => callback.call(this, ...args);
    this.#events[event2].push(cb);
    return () => {
      this.#events[event2] = this.#events[event2].filter(
        /** @param {any} fn */
        (fn) => fn !== cb
      );
    };
  }
  $destroy() {
    this.#instance.$destroy();
  }
};

// node_modules/svelte/src/internal/client/dom/elements/custom-element.js
var SvelteElement;
if (typeof HTMLElement === "function") {
  SvelteElement = class extends HTMLElement {
    /** The Svelte component constructor */
    $$ctor;
    /** Slots */
    $$s;
    /** @type {any} The Svelte component instance */
    $$c;
    /** Whether or not the custom element is connected */
    $$cn = false;
    /** @type {Record<string, any>} Component props data */
    $$d = {};
    /** `true` if currently in the process of reflecting component props back to attributes */
    $$r = false;
    /** @type {Record<string, CustomElementPropDefinition>} Props definition (name, reflected, type etc) */
    $$p_d = {};
    /** @type {Record<string, EventListenerOrEventListenerObject[]>} Event listeners */
    $$l = {};
    /** @type {Map<EventListenerOrEventListenerObject, Function>} Event listener unsubscribe functions */
    $$l_u = /* @__PURE__ */ new Map();
    /** @type {any} The managed render effect for reflecting attributes */
    $$me;
    /**
     * @param {*} $$componentCtor
     * @param {*} $$slots
     * @param {*} use_shadow_dom
     */
    constructor($$componentCtor, $$slots, use_shadow_dom) {
      super();
      this.$$ctor = $$componentCtor;
      this.$$s = $$slots;
      if (use_shadow_dom) {
        this.attachShadow({ mode: "open" });
      }
    }
    /**
     * @param {string} type
     * @param {EventListenerOrEventListenerObject} listener
     * @param {boolean | AddEventListenerOptions} [options]
     */
    addEventListener(type, listener, options) {
      this.$$l[type] = this.$$l[type] || [];
      this.$$l[type].push(listener);
      if (this.$$c) {
        const unsub = this.$$c.$on(type, listener);
        this.$$l_u.set(listener, unsub);
      }
      super.addEventListener(type, listener, options);
    }
    /**
     * @param {string} type
     * @param {EventListenerOrEventListenerObject} listener
     * @param {boolean | AddEventListenerOptions} [options]
     */
    removeEventListener(type, listener, options) {
      super.removeEventListener(type, listener, options);
      if (this.$$c) {
        const unsub = this.$$l_u.get(listener);
        if (unsub) {
          unsub();
          this.$$l_u.delete(listener);
        }
      }
    }
    async connectedCallback() {
      this.$$cn = true;
      if (!this.$$c) {
        let create_slot = function(name) {
          return (anchor) => {
            const slot2 = document.createElement("slot");
            if (name !== "default") slot2.name = name;
            append(anchor, slot2);
          };
        };
        await Promise.resolve();
        if (!this.$$cn || this.$$c) {
          return;
        }
        const $$slots = {};
        const existing_slots = get_custom_elements_slots(this);
        for (const name of this.$$s) {
          if (name in existing_slots) {
            if (name === "default" && !this.$$d.children) {
              this.$$d.children = create_slot(name);
              $$slots.default = true;
            } else {
              $$slots[name] = create_slot(name);
            }
          }
        }
        for (const attribute of this.attributes) {
          const name = this.$$g_p(attribute.name);
          if (!(name in this.$$d)) {
            this.$$d[name] = get_custom_element_value(name, attribute.value, this.$$p_d, "toProp");
          }
        }
        for (const key in this.$$p_d) {
          if (!(key in this.$$d) && this[key] !== void 0) {
            this.$$d[key] = this[key];
            delete this[key];
          }
        }
        this.$$c = createClassComponent({
          component: this.$$ctor,
          target: this.shadowRoot || this,
          props: {
            ...this.$$d,
            $$slots,
            $$host: this
          }
        });
        this.$$me = effect_root(() => {
          render_effect(() => {
            this.$$r = true;
            for (const key of object_keys(this.$$c)) {
              if (!this.$$p_d[key]?.reflect) continue;
              this.$$d[key] = this.$$c[key];
              const attribute_value = get_custom_element_value(
                key,
                this.$$d[key],
                this.$$p_d,
                "toAttribute"
              );
              if (attribute_value == null) {
                this.removeAttribute(this.$$p_d[key].attribute || key);
              } else {
                this.setAttribute(this.$$p_d[key].attribute || key, attribute_value);
              }
            }
            this.$$r = false;
          });
        });
        for (const type in this.$$l) {
          for (const listener of this.$$l[type]) {
            const unsub = this.$$c.$on(type, listener);
            this.$$l_u.set(listener, unsub);
          }
        }
        this.$$l = {};
      }
    }
    // We don't need this when working within Svelte code, but for compatibility of people using this outside of Svelte
    // and setting attributes through setAttribute etc, this is helpful
    /**
     * @param {string} attr
     * @param {string} _oldValue
     * @param {string} newValue
     */
    attributeChangedCallback(attr2, _oldValue, newValue) {
      if (this.$$r) return;
      attr2 = this.$$g_p(attr2);
      this.$$d[attr2] = get_custom_element_value(attr2, newValue, this.$$p_d, "toProp");
      this.$$c?.$set({ [attr2]: this.$$d[attr2] });
    }
    disconnectedCallback() {
      this.$$cn = false;
      Promise.resolve().then(() => {
        if (!this.$$cn && this.$$c) {
          this.$$c.$destroy();
          this.$$me();
          this.$$c = void 0;
        }
      });
    }
    /**
     * @param {string} attribute_name
     */
    $$g_p(attribute_name) {
      return object_keys(this.$$p_d).find(
        (key) => this.$$p_d[key].attribute === attribute_name || !this.$$p_d[key].attribute && key.toLowerCase() === attribute_name
      ) || attribute_name;
    }
  };
}
function get_custom_element_value(prop2, value, props_definition, transform) {
  const type = props_definition[prop2]?.type;
  value = type === "Boolean" && typeof value !== "boolean" ? value != null : value;
  if (!transform || !props_definition[prop2]) {
    return value;
  } else if (transform === "toAttribute") {
    switch (type) {
      case "Object":
      case "Array":
        return value == null ? null : JSON.stringify(value);
      case "Boolean":
        return value ? "" : null;
      case "Number":
        return value == null ? null : value;
      default:
        return value;
    }
  } else {
    switch (type) {
      case "Object":
      case "Array":
        return value && JSON.parse(value);
      case "Boolean":
        return value;
      // conversion already handled above
      case "Number":
        return value != null ? +value : value;
      default:
        return value;
    }
  }
}
function get_custom_elements_slots(element2) {
  const result = {};
  element2.childNodes.forEach((node) => {
    result[
      /** @type {Element} node */
      node.slot || "default"
    ] = true;
  });
  return result;
}

// node_modules/svelte/src/index-client.js
function onMount(fn) {
  if (component_context === null) {
    lifecycle_outside_component("onMount");
  }
  if (legacy_mode_flag && component_context.l !== null) {
    init_update_callbacks(component_context).m.push(fn);
  } else {
    user_effect(() => {
      const cleanup = untrack(fn);
      if (typeof cleanup === "function") return (
        /** @type {() => void} */
        cleanup
      );
    });
  }
}
function init_update_callbacks(context) {
  var l = (
    /** @type {ComponentContextLegacy} */
    context.l
  );
  return l.u ??= { a: [], b: [], m: [] };
}

// node_modules/svelte/src/version.js
var PUBLIC_VERSION = "5";

// node_modules/svelte/src/internal/disclose-version.js
if (typeof window !== "undefined")
  (window.__svelte ||= { v: /* @__PURE__ */ new Set() }).v.add(PUBLIC_VERSION);

// node_modules/svelte/src/internal/flags/legacy.js
enable_legacy_mode_flag();

// src/fava/adapter-client.ts
var PRIVATE_ADAPTER_BASE = "/__orangecount/fava";
function bootstrapPayload(wire, mtime = "") {
  const title = wire.options?.title?.trim() || "OrangeCount";
  const locale = wire.fava_options?.locale === "zh-CN" ? "zh-CN" : "en";
  return {
    ledger_title: title,
    locale,
    locales: ["en", "zh-CN"],
    theme: "system",
    routes: ["income_statement", "balance_sheet", "trial_balance", "journal", "query", "holdings", "commodities", "documents", "events", "statistics", "editor", "import", "options", "help", "account"],
    accounts: wire.accounts || [],
    currencies: wire.currencies || [],
    // The evaluator joins repeated operating_currency declarations into one
    // space-separated value, preserving declaration order.
    operating_currencies: (wire.options?.operating_currency || "").split(/\s+/).filter(Boolean),
    render_commas: (wire.options?.render_commas || "").toUpperCase() === "TRUE",
    errors: wire.errors || [],
    mtime
  };
}
function createAdapterClient(fetcher = fetch, base = PRIVATE_ADAPTER_BASE) {
  let lastMtime = "";
  async function get3(resource, query = {}) {
    const params = new URLSearchParams(query);
    const response = await fetcher(`${base}/${resource}${params.size ? `?${params}` : ""}`, {
      headers: { Accept: "application/json" }
    });
    const payload = await response.json();
    if (!response.ok) throw new Error(payload.error || `Adapter request failed (${response.status})`);
    if (payload.mtime) lastMtime = payload.mtime;
    return payload.data;
  }
  return {
    bootstrap: async () => {
      const wire = await get3("ledger_data");
      return bootstrapPayload(wire, lastMtime);
    },
    changed: async () => get3("changed", lastMtime ? { mtime: lastMtime } : {}),
    load: (route, query = {}) => {
      const treeRoutes = /* @__PURE__ */ new Set(["income_statement", "balance_sheet", "trial_balance"]);
      const directRoutes = /* @__PURE__ */ new Set(["options", "help", "diagnostics", "source", "editor", "import", "journal"]);
      const resource = treeRoutes.has(route) || directRoutes.has(route) ? route : route.startsWith("holdings_by_") ? "reports/holdings" : `reports/${route}`;
      return get3(resource, query);
    }
  };
}

// src/fava/components/ErrorBoundary.svelte
var root_1 = template(`<section class="state-panel error-panel" role="alert"><h2>Unable to load this local view</h2> <p> </p> <button type="button">Try again</button></section>`);
function ErrorBoundary($$anchor, $$props) {
  let message = prop($$props, "message", 8, null);
  let onRetry = prop($$props, "onRetry", 8);
  var fragment = comment();
  var node = first_child(fragment);
  {
    var consequent = ($$anchor2) => {
      var section = root_1();
      var p = sibling(child(section), 2);
      var text2 = child(p, true);
      reset(p);
      var button = sibling(p, 2);
      reset(section);
      template_effect(() => set_text(text2, message()));
      event("click", button, function(...$$args) {
        onRetry()?.apply(this, $$args);
      });
      append($$anchor2, section);
    };
    var alternate = ($$anchor2) => {
      var fragment_1 = comment();
      var node_1 = first_child(fragment_1);
      slot(node_1, $$props, "default", {}, null);
      append($$anchor2, fragment_1);
    };
    if_block(node, ($$render) => {
      if (message()) $$render(consequent);
      else $$render(alternate, false);
    });
  }
  append($$anchor, fragment);
}

// src/translations.ts
var translations = {
  en: {
    subtitle: "Read-only local ledger view.",
    language: "Language",
    theme: "Theme",
    conversion: "Conversion",
    interval: "Interval",
    overview: "Overview",
    accounts: "Accounts",
    journal: "Journal",
    trialBalance: "Trial balance",
    balanceSheet: "Balance sheet",
    incomeStatement: "Income statement",
    holdings: "Holdings",
    prices: "Prices",
    commodities: "Commodities",
    events: "Events",
    documents: "Documents",
    statistics: "Statistics",
    diagnostics: "Diagnostics",
    query: "Query",
    status: "Status",
    snapshot: "Snapshot",
    valid: "Valid",
    accountsCount: "Accounts",
    diagnosticsCount: "Diagnostics",
    publishedAt: "Published",
    yes: "yes",
    no: "no",
    loading: "Loading\u2026",
    unavailable: "No valid snapshot.",
    rows: "rows",
    columns: "columns",
    run: "Run query",
    queryHint: "SELECT account, balance FROM accounts ORDER BY account",
    empty: "No rows.",
    requestFailed: "Request failed",
    source: "Source",
    file: "File",
    content: "Content",
    sourceHint: "Browse files in the resolved include graph.",
    documentsHint: "Document attachments are served only from configured roots.",
    from: "From",
    to: "To",
    apply: "Apply",
    reset: "Reset",
    approximate: "approximate",
    exact: "Exact value",
    period: "Period",
    valuation: "Valuation",
    allPeriods: "All periods",
    monthly: "Monthly",
    quarterly: "Quarterly",
    yearly: "Yearly",
    atCost: "At cost",
    marketValue: "Market value",
    exportCSV: "Export CSV",
    chart: "Chart",
    tree: "Account tree",
    flag: "Flag",
    tag: "Tag",
    link: "Link",
    payee: "Payee",
    narration: "Narration",
    expand: "Expand details",
    save: "Save",
    saved: "Saved queries",
    queryName: "Query name",
    download: "Download",
    previewDiagnostics: "Preview diagnostics; commit will revalidate",
    previousPage: "Previous page",
    nextPage: "Next page",
    pageOf: "of",
    editor: "Editor",
    import: "Import",
    options: "Options",
    help: "Help",
    files: "Files",
    validate: "Validate",
    time: "Time",
    currency: "Currency",
    adapter: "Adapter",
    offset: "Offset",
    commit: "Commit",
    preview: "Preview",
    target: "Target ledger",
    chooseFile: "Choose a local file",
    backup: "Backup",
    discard: "Discard",
    syntax: "Syntax preview",
    noFile: "Select a file",
    searchHelp: "Search help",
    back: "Back",
    chartData: "Chart data",
    unavailablePrice: "Unavailable: no local price",
    unavailableCurrency: "Unavailable: no conversion quote",
    notValued: "Not valued"
  },
  "zh-CN": {
    subtitle: "\u53EA\u8BFB\u672C\u5730\u8D26\u672C\u89C6\u56FE\u3002",
    language: "\u8BED\u8A00",
    overview: "\u6982\u89C8",
    accounts: "\u8D26\u6237",
    journal: "\u65E5\u8BB0\u8D26",
    trialBalance: "\u8BD5\u7B97\u5E73\u8861",
    balanceSheet: "\u8D44\u4EA7\u8D1F\u503A\u8868",
    incomeStatement: "\u635F\u76CA\u8868",
    holdings: "\u6301\u4ED3",
    theme: "\u4E3B\u9898",
    conversion: "\u6362\u7B97",
    interval: "\u533A\u95F4",
    prices: "\u4EF7\u683C",
    commodities: "\u5546\u54C1",
    events: "\u4E8B\u4EF6",
    documents: "\u6587\u6863",
    statistics: "\u7EDF\u8BA1",
    diagnostics: "\u8BCA\u65AD",
    query: "\u67E5\u8BE2",
    status: "\u72B6\u6001",
    snapshot: "\u5FEB\u7167",
    valid: "\u6709\u6548",
    accountsCount: "\u8D26\u6237\u6570",
    diagnosticsCount: "\u8BCA\u65AD\u6570",
    publishedAt: "\u53D1\u5E03\u65F6\u95F4",
    yes: "\u662F",
    no: "\u5426",
    loading: "\u52A0\u8F7D\u4E2D\u2026",
    unavailable: "\u6CA1\u6709\u6709\u6548\u5FEB\u7167\u3002",
    rows: "\u884C",
    columns: "\u5217",
    run: "\u8FD0\u884C\u67E5\u8BE2",
    queryHint: "SELECT account, balance FROM accounts ORDER BY account",
    empty: "\u6CA1\u6709\u6570\u636E\u3002",
    requestFailed: "\u8BF7\u6C42\u5931\u8D25",
    source: "\u6E90\u6587\u4EF6",
    file: "\u6587\u4EF6",
    content: "\u5185\u5BB9",
    sourceHint: "\u6D4F\u89C8\u5DF2\u89E3\u6790 include \u56FE\u4E2D\u7684\u6587\u4EF6\u3002",
    documentsHint: "\u6587\u6863\u9644\u4EF6\u4EC5\u4ECE\u5DF2\u914D\u7F6E\u7684\u6839\u76EE\u5F55\u63D0\u4F9B\u3002",
    from: "\u8D77\u59CB\u65E5\u671F",
    to: "\u7ED3\u675F\u65E5\u671F",
    apply: "\u5E94\u7528",
    reset: "\u91CD\u7F6E",
    approximate: "\u8FD1\u4F3C\u503C",
    exact: "\u7CBE\u786E\u503C",
    period: "\u671F\u95F4",
    valuation: "\u4F30\u503C",
    allPeriods: "\u5168\u90E8\u671F\u95F4",
    monthly: "\u6309\u6708",
    quarterly: "\u6309\u5B63\u5EA6",
    yearly: "\u6309\u5E74",
    atCost: "\u6309\u6210\u672C",
    marketValue: "\u6309\u5E02\u503C",
    exportCSV: "\u5BFC\u51FA CSV",
    chart: "\u56FE\u8868",
    tree: "\u8D26\u6237\u6811",
    flag: "\u6807\u8BB0",
    tag: "\u6807\u7B7E",
    link: "\u94FE\u63A5",
    payee: "\u6536\u6B3E\u65B9",
    narration: "\u6458\u8981",
    expand: "\u5C55\u5F00\u8BE6\u60C5",
    save: "\u4FDD\u5B58",
    saved: "\u5DF2\u4FDD\u5B58\u67E5\u8BE2",
    queryName: "\u67E5\u8BE2\u540D\u79F0",
    download: "\u4E0B\u8F7D",
    previewDiagnostics: "\u9884\u89C8\u5B58\u5728\u8BCA\u65AD\uFF1B\u63D0\u4EA4\u65F6\u4F1A\u91CD\u65B0\u9A8C\u8BC1",
    previousPage: "\u4E0A\u4E00\u9875",
    nextPage: "\u4E0B\u4E00\u9875",
    pageOf: "/",
    editor: "\u7F16\u8F91\u5668",
    import: "\u5BFC\u5165",
    options: "\u9009\u9879",
    help: "\u5E2E\u52A9",
    files: "\u6587\u4EF6",
    validate: "\u9A8C\u8BC1",
    time: "\u65F6\u95F4",
    currency: "\u8D27\u5E01",
    adapter: "\u9002\u914D\u5668",
    offset: "\u62B5\u9500\u8D26\u6237",
    commit: "\u63D0\u4EA4",
    preview: "\u9884\u89C8",
    target: "\u76EE\u6807\u8D26\u672C",
    chooseFile: "\u9009\u62E9\u672C\u5730\u6587\u4EF6",
    backup: "\u5907\u4EFD",
    discard: "\u4E22\u5F03",
    syntax: "\u8BED\u6CD5\u9884\u89C8",
    noFile: "\u9009\u62E9\u6587\u4EF6",
    searchHelp: "\u641C\u7D22\u5E2E\u52A9",
    back: "\u8FD4\u56DE",
    chartData: "\u56FE\u8868\u6570\u636E",
    unavailablePrice: "\u4E0D\u53EF\u7528\uFF1A\u6CA1\u6709\u672C\u5730\u4EF7\u683C",
    unavailableCurrency: "\u4E0D\u53EF\u7528\uFF1A\u6CA1\u6709\u6362\u7B97\u62A5\u4EF7",
    notValued: "\u672A\u4F30\u503C"
  }
};

// src/fava/router.mjs
var ROUTES = Object.freeze([
  "income_statement",
  "balance_sheet",
  "trial_balance",
  "journal",
  "query",
  "holdings",
  "holdings_by_account",
  "holdings_by_currency",
  "holdings_by_root_account",
  "holdings_by_commodity",
  "commodities",
  "documents",
  "events",
  "statistics",
  "editor",
  "import",
  "options",
  "help",
  "source",
  "diagnostics",
  "errors"
]);
var PATHS = Object.freeze({
  income_statement: "/income_statement",
  balance_sheet: "/balance_sheet",
  trial_balance: "/trial_balance",
  journal: "/journal",
  query: "/query",
  holdings: "/holdings",
  holdings_by_account: "/holdings/by_account",
  holdings_by_currency: "/holdings/by_currency",
  holdings_by_root_account: "/holdings/by_root_account",
  holdings_by_commodity: "/holdings/by_commodity",
  commodities: "/commodities",
  documents: "/documents",
  events: "/events",
  statistics: "/statistics",
  editor: "/editor",
  import: "/import",
  options: "/options",
  help: "/help",
  source: "/source",
  diagnostics: "/diagnostics",
  errors: "/errors"
});
var QUERY_KEYS = Object.freeze(["time", "account", "filter", "conversion", "interval", "path", "query_string"]);
function pathWithoutTrailingSlash(pathname) {
  if (pathname.length > 1 && pathname.endsWith("/")) return pathname.slice(0, -1);
  return pathname || "/";
}
function decodeAccount(value) {
  try {
    return decodeURIComponent(value);
  } catch {
    return value;
  }
}
function parseRoute(input, basePath = "") {
  const url = input instanceof URL ? new URL(input.href) : new URL(input, "https://orange-count.invalid");
  let pathname = pathWithoutTrailingSlash(url.pathname);
  if (basePath && pathname.startsWith(basePath)) {
    pathname = pathWithoutTrailingSlash(pathname.slice(basePath.length));
  }
  let route = pathname === "/" ? "income_statement" : Object.entries(PATHS).find(([, path]) => path === pathname)?.[0] || "journal";
  if (pathname.startsWith("/help/")) route = "help";
  let account = "";
  const accountPrefix = "/account/";
  if (pathname.startsWith(accountPrefix)) {
    route = "account";
    account = decodeAccount(pathname.slice(accountPrefix.length));
  }
  const query = {};
  for (const key of QUERY_KEYS) {
    const value = url.searchParams.get(key);
    if (value) query[key] = value;
  }
  return { route, account, query, pathname };
}
function routeHref(route, { account = "", query = {} } = {}) {
  let pathname = PATHS[route] || "/journal";
  if (route === "account" && account) pathname = `/account/${encodeURIComponent(account)}`;
  const params = new URLSearchParams();
  for (const key of QUERY_KEYS) {
    if (query[key]) params.set(key, query[key]);
  }
  const suffix = params.toString();
  return `${pathname}${suffix ? `?${suffix}` : ""}`;
}
function updateQuery(current, changes) {
  const parsed = parseRoute(current);
  const query = { ...parsed.query };
  for (const key of QUERY_KEYS) {
    if (Object.hasOwn(changes, key)) {
      if (changes[key]) query[key] = changes[key];
      else delete query[key];
    }
  }
  return routeHref(parsed.route, { account: parsed.account, query });
}
function pageLabel(route) {
  const labels = {
    income_statement: "Income Statement",
    balance_sheet: "Balance Sheet",
    trial_balance: "Trial Balance",
    journal: "Journal",
    query: "Query",
    holdings: "Holdings",
    holdings_by_account: "Holdings by account",
    holdings_by_currency: "Holdings by currency",
    holdings_by_root_account: "Holdings by root account",
    holdings_by_commodity: "Holdings by commodity",
    commodities: "Commodities",
    documents: "Documents",
    events: "Events",
    statistics: "Statistics",
    editor: "Editor",
    import: "Import",
    options: "Options",
    help: "Help",
    source: "Source",
    diagnostics: "Diagnostics",
    errors: "Errors",
    account: "Account"
  };
  return labels[route] || "Journal";
}

// src/fava/components/PageTitle.svelte
var root_3 = template(`<a class="account-crumb svelte-95cklu"> </a><span class="crumb-sep svelte-95cklu">\u203A</span>`, 1);
var root = template(`<strong id="page-title" class="svelte-95cklu"><!></strong>`);
function PageTitle($$anchor, $$props) {
  push($$props, false);
  const catalog = mutable_state();
  const title = mutable_state();
  const segments = mutable_state();
  let route = prop($$props, "route", 8);
  let account = prop($$props, "account", 8, "");
  let locale = prop($$props, "locale", 8, "en");
  let onNavigate = prop($$props, "onNavigate", 8, () => {
  });
  const translationKeys = {
    income_statement: "incomeStatement",
    balance_sheet: "balanceSheet",
    trial_balance: "trialBalance",
    journal: "journal",
    query: "query",
    holdings: "holdings",
    commodities: "commodities",
    documents: "documents",
    events: "events",
    statistics: "statistics",
    editor: "editor",
    import: "import",
    options: "options",
    help: "help",
    source: "source",
    diagnostics: "diagnostics",
    errors: "diagnostics"
  };
  legacy_pre_effect(
    () => (translations, deep_read_state(locale())),
    () => {
      set(catalog, translations[locale() === "zh-CN" ? "zh-CN" : "en"]);
    }
  );
  legacy_pre_effect(
    () => (deep_read_state(route()), deep_read_state(account()), get(catalog), pageLabel),
    () => {
      set(title, route() === "account" && account() ? account() : get(catalog)[translationKeys[route()] || ""] || pageLabel(route()));
    }
  );
  legacy_pre_effect(
    () => (deep_read_state(route()), deep_read_state(account())),
    () => {
      set(segments, route() === "account" && account() ? account().split(":") : []);
    }
  );
  legacy_pre_effect_reset();
  init();
  var strong = root();
  var node = child(strong);
  {
    var consequent_1 = ($$anchor2) => {
      var fragment = comment();
      var node_1 = first_child(fragment);
      each(node_1, 1, () => get(segments), index, ($$anchor3, segment, index2) => {
        var fragment_1 = comment();
        var node_2 = first_child(fragment_1);
        {
          var consequent = ($$anchor4) => {
            var fragment_2 = root_3();
            var a = first_child(fragment_2);
            template_effect(() => set_attribute(a, "href", routeHref("account", {
              account: get(segments).slice(0, index2 + 1).join(":")
            })));
            var text2 = child(a, true);
            reset(a);
            next();
            template_effect(() => set_text(text2, get(segment)));
            event("click", a, preventDefault(() => onNavigate()(routeHref("account", {
              account: get(segments).slice(0, index2 + 1).join(":")
            }))));
            append($$anchor4, fragment_2);
          };
          var alternate = ($$anchor4) => {
            var text_1 = text();
            template_effect(() => set_text(text_1, get(segment)));
            append($$anchor4, text_1);
          };
          if_block(node_2, ($$render) => {
            if (index2 < get(segments).length - 1) $$render(consequent);
            else $$render(alternate, false);
          });
        }
        append($$anchor3, fragment_1);
      });
      append($$anchor2, fragment);
    };
    var alternate_1 = ($$anchor2) => {
      var text_2 = text();
      template_effect(() => set_text(text_2, get(title)));
      append($$anchor2, text_2);
    };
    if_block(node, ($$render) => {
      if (get(segments).length) $$render(consequent_1);
      else $$render(alternate_1, false);
    });
  }
  reset(strong);
  append($$anchor, strong);
  pop();
}

// src/fava/components/Header.svelte
var root2 = template(`<header><h1 class="svelte-ql43fz"><a class="ledger-title svelte-ql43fz" href="/"> </a> <!></h1> <span class="spacer svelte-ql43fz"></span> <form class="flex-row svelte-ql43fz" aria-label="Global filters"><input id="global-time" type="text" placeholder="Time" aria-label="Time"> <input id="global-account" type="text" placeholder="Account" aria-label="Account"> <input id="global-filter" type="text" placeholder="Filter by tag, payee, ..." aria-label="Filter by tag, payee, or narration"></form> <label class="header-select svelte-ql43fz"><span class="svelte-ql43fz"> </span> <select id="conversion" class="svelte-ql43fz"><option> </option><option> </option><option>Units</option><option> </option></select></label> <label class="header-select svelte-ql43fz"><span class="svelte-ql43fz"> </span> <select id="interval" class="svelte-ql43fz"><option> </option><option> </option><option> </option></select></label></header>`);
function Header($$anchor, $$props) {
  push($$props, false);
  let ledgerTitle = prop($$props, "ledgerTitle", 8);
  let route = prop($$props, "route", 8);
  let account = prop($$props, "account", 8, "");
  let locale = prop($$props, "locale", 8);
  let time = prop($$props, "time", 8, "");
  let accountFilter = prop($$props, "accountFilter", 8, "");
  let filter = prop($$props, "filter", 8, "");
  let conversion = prop($$props, "conversion", 8, "at_cost");
  let interval = prop($$props, "interval", 8, "month");
  let onNavigate = prop($$props, "onNavigate", 8);
  let onTime = prop($$props, "onTime", 8);
  let onAccount = prop($$props, "onAccount", 8);
  let onQuery = prop($$props, "onQuery", 8);
  let onConversion = prop($$props, "onConversion", 8);
  let onInterval = prop($$props, "onInterval", 8);
  function t(key) {
    return translations[locale() === "zh-CN" ? "zh-CN" : "en"][key] || translations.en[key] || key;
  }
  init();
  var header = root2();
  var h1 = child(header);
  var a = child(h1);
  var text2 = child(a, true);
  reset(a);
  var node = sibling(a, 2);
  PageTitle(node, {
    get route() {
      return route();
    },
    get account() {
      return account();
    },
    get locale() {
      return locale();
    },
    get onNavigate() {
      return onNavigate();
    }
  });
  reset(h1);
  var form = sibling(h1, 4);
  var input = child(form);
  remove_input_defaults(input);
  var input_1 = sibling(input, 2);
  remove_input_defaults(input_1);
  var input_2 = sibling(input_1, 2);
  remove_input_defaults(input_2);
  reset(form);
  var label = sibling(form, 2);
  var span = child(label);
  var text_1 = child(span, true);
  template_effect(() => set_text(text_1, t("conversion")));
  reset(span);
  var select = sibling(span, 2);
  init_select(select, conversion);
  var select_value;
  var option = child(select);
  option.value = null == (option.__value = "at_cost") ? "" : "at_cost";
  var text_2 = child(option, true);
  template_effect(() => set_text(text_2, t("atCost")));
  reset(option);
  var option_1 = sibling(option);
  option_1.value = null == (option_1.__value = "market_value") ? "" : "market_value";
  var text_3 = child(option_1, true);
  template_effect(() => set_text(text_3, t("marketValue")));
  reset(option_1);
  var option_2 = sibling(option_1);
  option_2.value = null == (option_2.__value = "units") ? "" : "units";
  var option_3 = sibling(option_2);
  option_3.value = null == (option_3.__value = "currency") ? "" : "currency";
  var text_4 = child(option_3, true);
  template_effect(() => set_text(text_4, t("currency")));
  reset(option_3);
  reset(select);
  reset(label);
  var label_1 = sibling(label, 2);
  var span_1 = child(label_1);
  var text_5 = child(span_1, true);
  template_effect(() => set_text(text_5, t("interval")));
  reset(span_1);
  var select_1 = sibling(span_1, 2);
  init_select(select_1, interval);
  var select_1_value;
  var option_4 = child(select_1);
  option_4.value = null == (option_4.__value = "month") ? "" : "month";
  var text_6 = child(option_4, true);
  template_effect(() => set_text(text_6, t("monthly")));
  reset(option_4);
  var option_5 = sibling(option_4);
  option_5.value = null == (option_5.__value = "quarter") ? "" : "quarter";
  var text_7 = child(option_5, true);
  template_effect(() => set_text(text_7, t("quarterly")));
  reset(option_5);
  var option_6 = sibling(option_5);
  option_6.value = null == (option_6.__value = "year") ? "" : "year";
  var text_8 = child(option_6, true);
  template_effect(() => set_text(text_8, t("yearly")));
  reset(option_6);
  reset(select_1);
  reset(label_1);
  reset(header);
  template_effect(() => {
    set_text(text2, ledgerTitle());
    set_value(input, time());
    set_value(input_1, accountFilter());
    set_value(input_2, filter());
    if (select_value !== (select_value = conversion())) {
      select.value = null == (select.__value = conversion()) ? "" : conversion(), select_option(select, conversion());
    }
    if (select_1_value !== (select_1_value = interval())) {
      select_1.value = null == (select_1.__value = interval()) ? "" : interval(), select_option(select_1, interval());
    }
  });
  event("click", a, preventDefault(() => onNavigate()("/")));
  event("change", input, (event2) => onTime()(event2.currentTarget.value));
  event("change", input_1, (event2) => onAccount()(event2.currentTarget.value));
  event("change", input_2, (event2) => onQuery()(event2.currentTarget.value));
  event("submit", form, preventDefault(function($$arg) {
    bubble_event.call(this, $$props, $$arg);
  }));
  event("change", select, (event2) => onConversion()(event2.currentTarget.value));
  event("change", select_1, (event2) => onInterval()(event2.currentTarget.value));
  append($$anchor, header);
  pop();
}

// src/fava/components/LoadingBoundary.svelte
var root_12 = template(`<div class="state-panel loading-panel" role="status" aria-live="polite"><span class="spinner" aria-hidden="true"></span> <span>Loading local view\u2026</span></div>`);
function LoadingBoundary($$anchor, $$props) {
  let active = prop($$props, "active", 8, false);
  var fragment = comment();
  var node = first_child(fragment);
  {
    var consequent = ($$anchor2) => {
      var div = root_12();
      append($$anchor2, div);
    };
    var alternate = ($$anchor2) => {
      var fragment_1 = comment();
      var node_1 = first_child(fragment_1);
      slot(node_1, $$props, "default", {}, null);
      append($$anchor2, fragment_1);
    };
    if_block(node, ($$render) => {
      if (active()) $$render(consequent);
      else $$render(alternate, false);
    });
  }
  append($$anchor, fragment);
}

// src/fava/reports/types.ts
function isRecord(value) {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}
function decimal(value) {
  return isRecord(value) && typeof value.display === "string" && typeof value.exact === "string" && typeof value.approximate === "boolean";
}
function decimalMap(value) {
  return isRecord(value) && Object.values(value).every(decimal);
}
function treeNode(value) {
  return isRecord(value) && typeof value.account === "string" && decimalMap(value.balance) && decimalMap(value.balance_children) && (value.cost === null || decimalMap(value.cost)) && (value.cost_children === null || decimalMap(value.cost_children)) && Array.isArray(value.children) && value.children.every(treeNode) && typeof value.has_txns === "boolean";
}
function chart(value) {
  if (!isRecord(value) || typeof value.kind !== "string" || typeof value.title !== "string" || typeof value.unit !== "string" || typeof value.currency !== "string" || typeof value.valuation !== "string" || typeof value.period !== "string" || typeof value.interval !== "string" || typeof value.measure !== "string" || !Array.isArray(value.series)) return false;
  return value.series.every((series) => {
    if (!isRecord(series) || typeof series.label !== "string" || !Array.isArray(series.points)) return false;
    return series.points.every((point) => isRecord(point) && typeof point.date === "string" && decimal(point.value));
  });
}
function parseTableReport(value) {
  if (!isRecord(value) || !Array.isArray(value.columns) || !value.columns.every((column) => typeof column === "string") || !Array.isArray(value.rows) || !value.rows.every(isRecord)) {
    throw new Error("Adapter returned an invalid table-report payload");
  }
  if (value.chart !== void 0 && value.chart !== null && !chart(value.chart)) {
    throw new Error("Adapter returned an invalid table-report chart");
  }
  return {
    columns: value.columns,
    rows: value.rows,
    chart: value.chart
  };
}
function parseTreeReport(value) {
  if (!isRecord(value) || !Array.isArray(value.trees) || !value.trees.every(treeNode) || !Array.isArray(value.charts) || !value.charts.every(chart)) {
    throw new Error("Adapter returned an invalid tree-report payload");
  }
  const dateRange = value.date_range;
  if (dateRange !== null && (!isRecord(dateRange) || typeof dateRange.begin !== "string" || typeof dateRange.end !== "string")) {
    throw new Error("Adapter returned an invalid report date range");
  }
  return {
    date_range: dateRange,
    charts: value.charts,
    trees: value.trees
  };
}
function formatAmount(value, group = false) {
  const display = value?.display ?? "";
  if (!group || display === "" || display.includes("/")) return display;
  const match = /^(-?)(\d+)(\.\d+)?$/.exec(display);
  if (!match) return display;
  const [, sign, integer, fraction = ""] = match;
  return `${sign}${integer.replace(/\B(?=(\d{3})+(?!\d))/g, ",")}${fraction}`;
}
function currenciesInTree(tree) {
  const values = /* @__PURE__ */ new Set();
  function visit(node) {
    Object.keys(node.balance).forEach((currency) => values.add(currency));
    Object.keys(node.balance_children).forEach((currency) => values.add(currency));
    node.children.forEach(visit);
  }
  visit(tree);
  return [...values].sort();
}
function parseJournalReport(value) {
  if (!isRecord(value) || !Array.isArray(value.entries) || !value.entries.every(isRecord)) {
    throw new Error("Adapter returned an invalid journal payload");
  }
  return { entries: value.entries };
}

// src/fava/charts/ReportChart.svelte
var root_32 = ns_template(`<rect opacity=".8"><title> </title></rect>`);
var root_2 = template(`<svg class="report-chart report-hierarchy-chart svelte-15jud14" viewBox="0 0 100 52" preserveAspectRatio="none" role="img"></svg>`);
var root_7 = ns_template(`<rect></rect>`);
var root_5 = template(`<svg class="report-chart report-bar-chart svelte-15jud14" viewBox="0 0 100 52" role="img"><line x1="2" x2="98" class="chart-axis svelte-15jud14"></line><!></svg>`);
var root_9 = ns_template(`<path class="svelte-15jud14"></path>`);
var root_8 = template(`<svg class="report-chart report-line-chart svelte-15jud14" viewBox="0 0 100 52" role="img"><line x1="2" x2="98" class="chart-axis svelte-15jud14"></line><!></svg>`);
var root_10 = template(`<span class="legend svelte-15jud14"><i class="svelte-15jud14"></i> </span>`);
var root_11 = template(`<p class="chart-availability svelte-15jud14"> </p>`);
var root_122 = template(`<th scope="col" class="num"> </th>`);
var root_14 = template(`<td class="num"> </td>`);
var root_13 = template(`<tr><th scope="row"> </th><!></tr>`);
var root_15 = template(`<tr><td>No chart data.</td></tr>`);
var root_16 = template(`<section class="chart-card svelte-15jud14"><h3> </h3> <!> <p class="chart-meta svelte-15jud14"> </p> <!> <!> <details class="chart-data svelte-15jud14"><summary class="svelte-15jud14"> </summary> <div class="chart-scroll svelte-15jud14"><table class="svelte-15jud14"><thead><tr><th scope="col">Period</th><!></tr></thead><tbody></tbody></table></div></details></section>`);
function ReportChart($$anchor, $$props) {
  push($$props, false);
  const catalog = mutable_state();
  const availabilityKey = mutable_state();
  const availabilityText = mutable_state();
  const pointValues = mutable_state();
  const min = mutable_state();
  const max = mutable_state();
  const range = mutable_state();
  const width = mutable_state();
  const leaves = mutable_state();
  const roots = mutable_state();
  const tiles = mutable_state();
  let chart2 = prop($$props, "chart", 8);
  let locale = prop($$props, "locale", 8, "en");
  function label(key) {
    return get(catalog)[key] ?? translations.en[key] ?? key;
  }
  const availabilityKeys = {
    "unavailable-price": "unavailablePrice",
    "unavailable-currency": "unavailableCurrency"
  };
  const colors = [
    "var(--series-0, #2563eb)",
    "var(--series-1, #d97706)",
    "var(--series-2, #16a34a)",
    "var(--series-3, #9333ea)"
  ];
  function numberValue(value) {
    if (value.display.includes("/")) {
      const [numerator, denominator] = value.display.split("/").map(Number);
      return denominator ? numerator / denominator : 0;
    }
    const parsed = Number(value.display);
    return Number.isFinite(parsed) ? parsed : 0;
  }
  function x(index2) {
    return get(width) === 1 ? 50 : index2 / (get(width) - 1) * 96 + 2;
  }
  function y(value) {
    return 48 - (value - get(min)) / get(range) * 44;
  }
  function linePath(points) {
    return points.map((point, index2) => `${index2 ? "L" : "M"}${x(index2).toFixed(2)},${y(numberValue(point.value)).toFixed(2)}`).join(" ");
  }
  function barHeight(value) {
    return Math.max(0.5, Math.abs(y(value) - y(0)));
  }
  function barY(value) {
    return value >= 0 ? y(value) : y(0);
  }
  function collectLeaves(nodes, root13 = "") {
    const out = [];
    for (const node of nodes) {
      const top = root13 || node.name.split(":")[0];
      if (node.children?.length) {
        out.push(...collectLeaves(node.children, top));
      } else {
        out.push({
          name: node.name,
          root: top,
          value: Math.abs(numberValue(node.value)),
          display: node.value.display
        });
      }
    }
    return out;
  }
  function tile(items, x0, y0, w, h) {
    if (!items.length || w <= 0 || h <= 0) return [];
    if (items.length === 1) return [{ ...items[0], x: x0, y: y0, w, h }];
    const total = items.reduce((sum, item) => sum + item.value, 0);
    if (!total) return [];
    let running = 0;
    let split = 1;
    for (let index2 = 0; index2 < items.length; index2 += 1) {
      running += items[index2].value;
      if (running >= total / 2) {
        split = index2 + 1;
        break;
      }
    }
    split = Math.min(Math.max(split, 1), items.length - 1);
    const head2 = items.slice(0, split);
    const tail = items.slice(split);
    const share = head2.reduce((sum, item) => sum + item.value, 0) / total;
    if (w >= h) {
      const headWidth = w * share;
      return [
        ...tile(head2, x0, y0, headWidth, h),
        ...tile(tail, x0 + headWidth, y0, w - headWidth, h)
      ];
    }
    const headHeight = h * share;
    return [
      ...tile(head2, x0, y0, w, headHeight),
      ...tile(tail, x0, y0 + headHeight, w, h - headHeight)
    ];
  }
  function tileColor(leaf) {
    return colors[get(roots).indexOf(leaf.root) % colors.length];
  }
  legacy_pre_effect(
    () => (translations, deep_read_state(locale())),
    () => {
      set(catalog, translations[locale() === "zh-CN" ? "zh-CN" : "en"]);
    }
  );
  legacy_pre_effect(() => deep_read_state(chart2()), () => {
    set(availabilityKey, chart2().availability ? availabilityKeys[chart2().availability] ?? "" : "");
  });
  legacy_pre_effect(() => get(availabilityKey), () => {
    set(availabilityText, get(availabilityKey) ? label(get(availabilityKey)) : "");
  });
  legacy_pre_effect(() => deep_read_state(chart2()), () => {
    set(pointValues, chart2().series.flatMap((series) => series.points.map((point) => numberValue(point.value))));
  });
  legacy_pre_effect(() => get(pointValues), () => {
    set(min, Math.min(0, ...get(pointValues).length ? get(pointValues) : [0]));
  });
  legacy_pre_effect(() => get(pointValues), () => {
    set(max, Math.max(0, ...get(pointValues).length ? get(pointValues) : [0]));
  });
  legacy_pre_effect(() => (get(max), get(min)), () => {
    set(range, get(max) - get(min) || 1);
  });
  legacy_pre_effect(() => deep_read_state(chart2()), () => {
    set(width, Math.max(1, chart2().series[0]?.points.length ?? 1));
  });
  legacy_pre_effect(() => deep_read_state(chart2()), () => {
    set(leaves, (chart2().nodes ?? []).flatMap((node) => collectLeaves([node])).filter((leaf) => leaf.value > 0).sort((left, right) => right.value - left.value));
  });
  legacy_pre_effect(() => get(leaves), () => {
    set(roots, [
      ...new Set(get(leaves).map((leaf) => leaf.root))
    ].sort());
  });
  legacy_pre_effect(() => get(leaves), () => {
    set(tiles, tile(get(leaves), 0, 0, 100, 52));
  });
  legacy_pre_effect_reset();
  init();
  var section = root_16();
  var h3 = child(section);
  var text2 = child(h3, true);
  reset(h3);
  var node_1 = sibling(h3, 2);
  {
    var consequent = ($$anchor2) => {
      var svg = root_2();
      each(svg, 5, () => get(tiles), (item) => item.name, ($$anchor3, item) => {
        var rect = root_32();
        template_effect(() => set_attribute(rect, "width", Math.max(0.3, get(item).w - 0.3)));
        template_effect(() => set_attribute(rect, "height", Math.max(0.3, get(item).h - 0.3)));
        const style_derived = derived_safe_equal(() => `fill:${tileColor(get(item))}`);
        var title = child(rect);
        var text_1 = child(title);
        reset(title);
        reset(rect);
        template_effect(() => {
          set_attribute(rect, "x", get(item).x + 0.15);
          set_attribute(rect, "y", get(item).y + 0.15);
          set_attribute(rect, "style", get(style_derived));
          set_text(text_1, `${get(item).name ?? ""}: ${get(item).display ?? ""} ${chart2().currency ?? ""}`);
        });
        append($$anchor3, rect);
      });
      reset(svg);
      template_effect(() => set_attribute(svg, "aria-label", chart2().title));
      append($$anchor2, svg);
    };
    var alternate_1 = ($$anchor2) => {
      var fragment = comment();
      var node_2 = first_child(fragment);
      {
        var consequent_1 = ($$anchor3) => {
          var svg_1 = root_5();
          var line = child(svg_1);
          template_effect(() => set_attribute(line, "y1", y(0)));
          template_effect(() => set_attribute(line, "y2", y(0)));
          var node_3 = sibling(line);
          each(node_3, 3, () => chart2().series, (series) => series.label, ($$anchor4, series, seriesIndex) => {
            var fragment_1 = comment();
            var node_4 = first_child(fragment_1);
            each(node_4, 3, () => get(series).points, (point) => point.date, ($$anchor5, point, index2) => {
              var rect_1 = root_7();
              const value = derived_safe_equal(() => numberValue(get(point).value));
              const barWidth = derived_safe_equal(() => Math.max(1, 90 / Math.max(1, get(width)) / Math.max(1, chart2().series.length)));
              template_effect(() => set_attribute(rect_1, "x", x(get(index2)) - 45 / Math.max(1, get(width)) + get(seriesIndex) * get(barWidth)));
              template_effect(() => set_attribute(rect_1, "y", barY(get(value))));
              template_effect(() => set_attribute(rect_1, "height", barHeight(get(value))));
              template_effect(() => {
                set_attribute(rect_1, "width", get(barWidth) - 0.25);
                set_attribute(rect_1, "style", `fill:${colors[get(seriesIndex) % colors.length]}`);
              });
              append($$anchor5, rect_1);
            });
            append($$anchor4, fragment_1);
          });
          reset(svg_1);
          template_effect(() => set_attribute(svg_1, "aria-label", chart2().title));
          append($$anchor3, svg_1);
        };
        var alternate = ($$anchor3) => {
          var svg_2 = root_8();
          var line_1 = child(svg_2);
          template_effect(() => set_attribute(line_1, "y1", y(0)));
          template_effect(() => set_attribute(line_1, "y2", y(0)));
          var node_5 = sibling(line_1);
          each(node_5, 3, () => chart2().series, (series) => series.label, ($$anchor4, series, index2) => {
            var path = root_9();
            template_effect(() => set_attribute(path, "d", linePath(get(series).points)));
            template_effect(() => set_attribute(path, "style", `stroke:${colors[get(index2) % colors.length]}`));
            append($$anchor4, path);
          });
          reset(svg_2);
          template_effect(() => set_attribute(svg_2, "aria-label", chart2().title));
          append($$anchor3, svg_2);
        };
        if_block(
          node_2,
          ($$render) => {
            if (chart2().kind === "stacked-bar" || chart2().kind === "bar") $$render(consequent_1);
            else $$render(alternate, false);
          },
          true
        );
      }
      append($$anchor2, fragment);
    };
    if_block(node_1, ($$render) => {
      if (chart2().kind === "hierarchy" && get(tiles).length) $$render(consequent);
      else $$render(alternate_1, false);
    });
  }
  var p = sibling(node_1, 2);
  var text_2 = child(p);
  reset(p);
  var node_6 = sibling(p, 2);
  each(node_6, 3, () => chart2().series, (series) => series.label, ($$anchor2, series, index2) => {
    var span = root_10();
    var i = child(span);
    var text_3 = sibling(i, 1, true);
    reset(span);
    template_effect(() => {
      set_attribute(i, "style", `background:${colors[get(index2) % colors.length]}`);
      set_text(text_3, get(series).label);
    });
    append($$anchor2, span);
  });
  var node_7 = sibling(node_6, 2);
  {
    var consequent_2 = ($$anchor2) => {
      var p_1 = root_11();
      var text_4 = child(p_1, true);
      reset(p_1);
      template_effect(() => set_text(text_4, get(availabilityText)));
      append($$anchor2, p_1);
    };
    if_block(node_7, ($$render) => {
      if (get(availabilityText)) $$render(consequent_2);
    });
  }
  var details = sibling(node_7, 2);
  var summary = child(details);
  var text_5 = child(summary, true);
  template_effect(() => set_text(text_5, label("chartData")));
  reset(summary);
  var div = sibling(summary, 2);
  var table = child(div);
  var thead = child(table);
  var tr = child(thead);
  var node_8 = sibling(child(tr));
  each(node_8, 1, () => chart2().series, (series) => series.label, ($$anchor2, series) => {
    var th = root_122();
    var text_6 = child(th, true);
    reset(th);
    template_effect(() => set_text(text_6, get(series).label));
    append($$anchor2, th);
  });
  reset(tr);
  reset(thead);
  var tbody = sibling(thead);
  each(
    tbody,
    7,
    () => chart2().series[0]?.points ?? [],
    (point) => point.date,
    ($$anchor2, point, index2) => {
      var tr_1 = root_13();
      var th_1 = child(tr_1);
      var text_7 = child(th_1, true);
      reset(th_1);
      var node_9 = sibling(th_1);
      each(node_9, 1, () => chart2().series, (series) => series.label, ($$anchor3, series) => {
        var td = root_14();
        var text_8 = child(td, true);
        template_effect(() => set_text(text_8, formatAmount(get(series).points[get(index2)]?.value)));
        reset(td);
        append($$anchor3, td);
      });
      reset(tr_1);
      template_effect(() => set_text(text_7, get(point).date));
      append($$anchor2, tr_1);
    },
    ($$anchor2) => {
      var tr_2 = root_15();
      var td_1 = child(tr_2);
      reset(tr_2);
      template_effect(() => set_attribute(td_1, "colspan", chart2().series.length + 1));
      append($$anchor2, tr_2);
    }
  );
  reset(tbody);
  reset(table);
  reset(div);
  reset(details);
  reset(section);
  template_effect(() => {
    set_attribute(section, "aria-label", chart2().title);
    set_text(text2, chart2().title);
    set_text(text_2, `${chart2().interval ?? ""} \xB7 ${chart2().valuation ?? ""}${(chart2().currency ? ` \xB7 ${chart2().currency}` : "") ?? ""}`);
  });
  append($$anchor, section);
  pop();
}

// src/fava/reports/GenericReport.svelte
var root_17 = template(`<a class="button">Export CSV</a>`);
var root_33 = template(`<th scope="col" class="svelte-1ssax5s"> </th>`);
var root_6 = template(`<a> </a>`);
var root_82 = template(`<a> </a>`);
var root_52 = template(`<td class="svelte-1ssax5s"><!></td>`);
var root_4 = template(`<tr></tr>`);
var root_102 = template(`<tr><td>No rows.</td></tr>`);
var root3 = template(`<div class="headerline"><h2> </h2> <span class="muted svelte-1ssax5s"> </span> <!></div> <!> <div class="table-scroll svelte-1ssax5s"><table class="report-table svelte-1ssax5s"><thead><tr></tr></thead><tbody></tbody></table></div>`, 1);
function GenericReport($$anchor, $$props) {
  push($$props, false);
  let report = prop($$props, "report", 8);
  let title = prop($$props, "title", 8);
  let route = prop($$props, "route", 8, "");
  let locale = prop($$props, "locale", 8, "en");
  let renderCommas = prop($$props, "renderCommas", 8, false);
  function display(value) {
    if (value && typeof value === "object" && "display" in value && typeof value.display === "string") {
      return formatAmount(value, renderCommas());
    }
    if (Array.isArray(value)) return value.join(", ");
    if (value && typeof value === "object") return JSON.stringify(value);
    return value == null ? "" : String(value);
  }
  function isNumberLike(value) {
    return typeof value === "number" || typeof value === "object" && value !== null && "display" in value;
  }
  init();
  var fragment = root3();
  var div = first_child(fragment);
  var h2 = child(div);
  var text2 = child(h2, true);
  reset(h2);
  var span = sibling(h2, 2);
  var text_1 = child(span);
  reset(span);
  var node = sibling(span, 2);
  {
    var consequent = ($$anchor2) => {
      var a = root_17();
      template_effect(() => set_attribute(a, "href", `/api/v1/reports/${route()}?format=csv`));
      append($$anchor2, a);
    };
    if_block(node, ($$render) => {
      if (route()) $$render(consequent);
    });
  }
  reset(div);
  var node_1 = sibling(div, 2);
  {
    var consequent_1 = ($$anchor2) => {
      ReportChart($$anchor2, {
        get chart() {
          return report().chart;
        },
        get locale() {
          return locale();
        }
      });
    };
    if_block(node_1, ($$render) => {
      if (report().chart) $$render(consequent_1);
    });
  }
  var div_1 = sibling(node_1, 2);
  var table = child(div_1);
  var thead = child(table);
  var tr = child(thead);
  each(tr, 5, () => report().columns, (column) => column, ($$anchor2, column) => {
    var th = root_33();
    const class_directive = derived_safe_equal(() => report().rows.some((row) => isNumberLike(row[get(column)])));
    template_effect(() => toggle_class(th, "num", get(class_directive)));
    var text_2 = child(th, true);
    reset(th);
    template_effect(() => set_text(text_2, get(column)));
    append($$anchor2, th);
  });
  reset(tr);
  reset(thead);
  var tbody = sibling(thead);
  each(
    tbody,
    5,
    () => report().rows,
    index,
    ($$anchor2, row) => {
      var tr_1 = root_4();
      each(tr_1, 5, () => report().columns, (column) => column, ($$anchor3, column) => {
        var td = root_52();
        const class_directive_1 = derived_safe_equal(() => isNumberLike(get(row)[get(column)]));
        template_effect(() => toggle_class(td, "num", get(class_directive_1)));
        var node_2 = child(td);
        {
          var consequent_2 = ($$anchor4) => {
            var a_1 = root_6();
            template_effect(() => set_attribute(a_1, "href", `/documents/${encodeURIComponent(get(row)[get(column)])}`));
            var text_3 = child(a_1, true);
            template_effect(() => set_text(text_3, display(get(row)[get(column)])));
            reset(a_1);
            append($$anchor4, a_1);
          };
          var alternate_1 = ($$anchor4) => {
            var fragment_2 = comment();
            var node_3 = first_child(fragment_2);
            {
              var consequent_3 = ($$anchor5) => {
                var a_2 = root_82();
                template_effect(() => set_attribute(a_2, "href", `/source?path=${encodeURIComponent(get(row)[get(column)])}`));
                var text_4 = child(a_2, true);
                template_effect(() => set_text(text_4, display(get(row)[get(column)])));
                reset(a_2);
                append($$anchor5, a_2);
              };
              var alternate = ($$anchor5) => {
                var text_5 = text();
                template_effect(() => set_text(text_5, display(get(row)[get(column)])));
                append($$anchor5, text_5);
              };
              if_block(
                node_3,
                ($$render) => {
                  if (["file", "path"].includes(get(column)) && typeof get(row)[get(column)] === "string") $$render(consequent_3);
                  else $$render(alternate, false);
                },
                true
              );
            }
            append($$anchor4, fragment_2);
          };
          if_block(node_2, ($$render) => {
            if (route() === "documents" && get(column) === "filename" && typeof get(row)[get(column)] === "string") $$render(consequent_2);
            else $$render(alternate_1, false);
          });
        }
        reset(td);
        append($$anchor3, td);
      });
      reset(tr_1);
      append($$anchor2, tr_1);
    },
    ($$anchor2) => {
      var tr_2 = root_102();
      var td_1 = child(tr_2);
      reset(tr_2);
      template_effect(() => set_attribute(td_1, "colspan", report().columns.length));
      append($$anchor2, tr_2);
    }
  );
  reset(tbody);
  reset(table);
  reset(div_1);
  template_effect(() => {
    set_text(text2, title());
    set_text(text_1, `${report().rows.length ?? ""} rows`);
  });
  append($$anchor, fragment);
  pop();
}

// src/fava/reports/JournalReport.svelte
var on_click = (_, toggle, chip) => toggle(get(chip).cls);
var root_18 = template(`<button type="button"> </button>`);
var root_34 = template(`<a> </a>`);
var root_42 = template(`<strong class="payee"> </strong><span class="separator"></span>`, 1);
var root_53 = template(`<span class="tag"> </span>`);
var root_62 = template(`<span class="link"> </span>`);
var root_72 = template(`<span class="filename"> </span>`);
var on_click_1 = (__1, toggleEntry, entry) => toggleEntry(get(entry));
var on_keydown = (event2, toggleEntry, entry) => {
  if (event2.key === "Enter" || event2.key === " ") {
    event2.preventDefault();
    toggleEntry(get(entry));
  }
};
var root_92 = template(`<span></span>`);
var root_83 = template(`<span class="indicators" role="button" tabindex="0" title="Toggle postings"></span>`);
var root_103 = template(`<span class="indicators"></span>`);
var root_112 = template(`<span class="num bal"> </span> <span class="change num"></span> <span class="change num"></span>`, 1);
var root_132 = template(`<span class="num"></span> <span class="num"></span> <span class="num"> </span>`, 1);
var root_152 = template(`<li><p><span class="datecell"></span> <span class="flag"> </span> <span class="description"><a> </a></span> <span class="num"> </span> <span class="num"> </span> <span class="num"> </span></p></li>`);
var root_142 = template(`<ul class="postings"></ul>`);
var root_172 = template(`<dt> </dt> <dd> </dd>`, 1);
var root_162 = template(`<dl class="metadata"></dl>`);
var root_22 = template(`<li><p><span class="datecell"> </span> <span class="flag"> </span> <span class="description"><!> <!> <!> <!> <!></span> <!> <!></p> <!> <!></li>`);
var root4 = template(`<form class="flex-row journal-chips svelte-3l7lkh"><!> <span class="spacer svelte-3l7lkh"></span> <a class="button" href="/api/v1/reports/journal?format=csv">Export CSV</a></form> <ol><li class="head"><p><span class="datecell">Date</span> <span class="flag">F</span> <span class="description">Payee/Narration</span> <span class="num">Units</span> <span class="num">Cost</span> <span class="num"> </span></p></li> <!></ol>`, 1);
function JournalReport($$anchor, $$props) {
  push($$props, false);
  const listClasses = mutable_state();
  let report = prop($$props, "report", 8);
  let renderCommas = prop($$props, "renderCommas", 8, false);
  let runningBalances = prop($$props, "runningBalances", 8, null);
  const chips = [
    {
      label: "Open",
      cls: "show-open",
      title: "Toggle Open entries"
    },
    {
      label: "Close",
      cls: "show-close",
      title: "Toggle Close entries"
    },
    {
      label: "Transaction",
      cls: "show-transaction",
      title: "Toggle Transaction entries"
    },
    {
      label: "*",
      cls: "show-cleared",
      title: "Cleared transactions"
    },
    {
      label: "!",
      cls: "show-pending",
      title: "Pending transactions"
    },
    {
      label: "x",
      cls: "show-other",
      title: "Other transactions"
    },
    {
      label: "Balance",
      cls: "show-balance",
      title: "Toggle Balance entries"
    },
    {
      label: "Note",
      cls: "show-note",
      title: "Toggle Note entries"
    },
    {
      label: "Document",
      cls: "show-document",
      title: "Toggle Document entries"
    },
    {
      label: "Pad",
      cls: "show-pad",
      title: "Toggle Pad entries"
    },
    {
      label: "Query",
      cls: "show-query",
      title: "Toggle Query entries"
    },
    {
      label: "Custom",
      cls: "show-custom",
      title: "Toggle Custom entries"
    },
    {
      label: "Metadata",
      cls: "show-metadata",
      title: "Toggle metadata"
    },
    {
      label: "Postings",
      cls: "show-postings",
      title: "Toggle postings"
    }
  ];
  let active = mutable_state(/* @__PURE__ */ new Set([
    "show-transaction",
    "show-cleared",
    "show-pending",
    "show-balance",
    "show-note",
    "show-document",
    "show-query",
    "show-custom"
  ]));
  function toggle(cls) {
    const next2 = new Set(get(active));
    if (next2.has(cls)) next2.delete(cls);
    else next2.add(cls);
    set(active, next2);
  }
  let expanded = mutable_state(/* @__PURE__ */ new Set());
  function toggleEntry(entry) {
    const next2 = new Set(get(expanded));
    if (next2.has(entry)) next2.delete(entry);
    else next2.add(entry);
    set(expanded, next2);
  }
  function flagClass(entry) {
    if (entry.type !== "transaction") return "";
    if (entry.flag === "*") return "cleared";
    if (entry.flag === "!") return "pending";
    return "other";
  }
  function amountText(amount) {
    if (!amount) return "";
    return `${formatAmount(amount.number, renderCommas())} ${amount.currency}`;
  }
  function accountHref(account) {
    return `/account/${encodeURIComponent(account)}`;
  }
  function describe(entry) {
    switch (entry.type) {
      case "open":
        return entry.extra?.currencies ?? "";
      case "pad":
        return entry.extra?.source_account ?? "";
      case "query":
        return entry.extra?.query ?? "";
      case "custom":
        return entry.extra?.values ?? "";
      case "event":
        return entry.extra?.value ?? "";
      default:
        return entry.narration ?? "";
    }
  }
  legacy_pre_effect(() => get(active), () => {
    set(listClasses, [
      "flex-table",
      "journal",
      ...get(active)
    ].join(" "));
  });
  legacy_pre_effect_reset();
  init();
  var fragment = root4();
  var form = first_child(fragment);
  var node = child(form);
  each(node, 1, () => chips, (chip) => chip.cls, ($$anchor2, chip) => {
    var button = root_18();
    template_effect(() => set_attribute(button, "aria-pressed", get(active).has(get(chip).cls)));
    button.__click = [on_click, toggle, chip];
    const class_directive = derived_safe_equal(() => !get(active).has(get(chip).cls));
    var text2 = child(button, true);
    reset(button);
    template_effect(() => {
      set_attribute(button, "title", get(chip).title);
      toggle_class(button, "inactive", get(class_directive));
      set_text(text2, get(chip).label);
    });
    append($$anchor2, button);
  });
  next(4);
  reset(form);
  var ol = sibling(form, 2);
  var li = child(ol);
  var p = child(li);
  var span = sibling(child(p), 10);
  var text_1 = child(span, true);
  reset(span);
  reset(p);
  reset(li);
  var node_1 = sibling(li, 2);
  each(node_1, 3, () => report().entries, (entry, index2) => entry.type + entry.date + index2, ($$anchor2, entry) => {
    var li_1 = root_22();
    const class_derived = derived_safe_equal(() => `${get(entry).type ?? ""} ${flagClass(get(entry)) ?? ""} svelte-3l7lkh`);
    const class_directive_1 = derived_safe_equal(() => get(expanded).has(get(entry)));
    var p_1 = child(li_1);
    var span_1 = child(p_1);
    var text_2 = child(span_1, true);
    reset(span_1);
    var span_2 = sibling(span_1, 2);
    var text_3 = child(span_2, true);
    reset(span_2);
    var span_3 = sibling(span_2, 2);
    var node_2 = child(span_3);
    {
      var consequent = ($$anchor3) => {
        var a = root_34();
        template_effect(() => set_attribute(a, "href", accountHref(get(entry).account)));
        var text_4 = child(a, true);
        reset(a);
        template_effect(() => set_text(text_4, get(entry).account));
        append($$anchor3, a);
      };
      if_block(node_2, ($$render) => {
        if (get(entry).account) $$render(consequent);
      });
    }
    var node_3 = sibling(node_2, 2);
    {
      var consequent_1 = ($$anchor3) => {
        var fragment_1 = root_42();
        var strong = first_child(fragment_1);
        var text_5 = child(strong, true);
        reset(strong);
        next();
        template_effect(() => set_text(text_5, get(entry).payee));
        append($$anchor3, fragment_1);
      };
      if_block(node_3, ($$render) => {
        if (get(entry).payee) $$render(consequent_1);
      });
    }
    var text_6 = sibling(node_3);
    template_effect(() => set_text(text_6, ` ${describe(get(entry)) ?? ""} `));
    var node_4 = sibling(text_6);
    each(node_4, 1, () => get(entry).tags ?? [], (tag) => tag, ($$anchor3, tag) => {
      var span_4 = root_53();
      var text_7 = child(span_4);
      reset(span_4);
      template_effect(() => set_text(text_7, `#${get(tag) ?? ""}`));
      append($$anchor3, span_4);
    });
    var node_5 = sibling(node_4, 2);
    each(node_5, 1, () => get(entry).links ?? [], (link2) => link2, ($$anchor3, link2) => {
      var span_5 = root_62();
      var text_8 = child(span_5);
      reset(span_5);
      template_effect(() => set_text(text_8, `^${get(link2) ?? ""}`));
      append($$anchor3, span_5);
    });
    var node_6 = sibling(node_5, 2);
    each(node_6, 1, () => get(entry).filenames ?? [], (filename) => filename, ($$anchor3, filename) => {
      var span_6 = root_72();
      var text_9 = child(span_6, true);
      reset(span_6);
      template_effect(() => set_text(text_9, get(filename)));
      append($$anchor3, span_6);
    });
    reset(span_3);
    var node_7 = sibling(span_3, 2);
    {
      var consequent_2 = ($$anchor3) => {
        var span_7 = root_83();
        span_7.__click = [on_click_1, toggleEntry, entry];
        span_7.__keydown = [on_keydown, toggleEntry, entry];
        each(span_7, 5, () => get(entry).postings, index, ($$anchor4, posting) => {
          var span_8 = root_92();
          template_effect(() => set_class(span_8, get(posting).flag === "!" ? "pending" : ""));
          append($$anchor4, span_8);
        });
        reset(span_7);
        append($$anchor3, span_7);
      };
      var alternate = ($$anchor3) => {
        var span_9 = root_103();
        append($$anchor3, span_9);
      };
      if_block(node_7, ($$render) => {
        if (get(entry).postings?.length) $$render(consequent_2);
        else $$render(alternate, false);
      });
    }
    var node_8 = sibling(node_7, 2);
    {
      var consequent_3 = ($$anchor3) => {
        var fragment_2 = root_112();
        var span_10 = first_child(fragment_2);
        var text_10 = child(span_10, true);
        template_effect(() => set_text(text_10, amountText(get(entry).amount)));
        reset(span_10);
        next(4);
        template_effect(() => set_attribute(span_10, "title", get(entry).amount.currency));
        append($$anchor3, fragment_2);
      };
      var alternate_1 = ($$anchor3) => {
        var fragment_3 = comment();
        var node_9 = first_child(fragment_3);
        {
          var consequent_4 = ($$anchor4) => {
            var fragment_4 = root_132();
            var span_11 = sibling(first_child(fragment_4), 4);
            var text_11 = child(span_11, true);
            template_effect(() => set_text(text_11, runningBalances().get(get(entry)) ?? ""));
            reset(span_11);
            append($$anchor4, fragment_4);
          };
          if_block(
            node_9,
            ($$render) => {
              if (runningBalances()) $$render(consequent_4);
            },
            true
          );
        }
        append($$anchor3, fragment_3);
      };
      if_block(node_8, ($$render) => {
        if (get(entry).amount) $$render(consequent_3);
        else $$render(alternate_1, false);
      });
    }
    reset(p_1);
    var node_10 = sibling(p_1, 2);
    {
      var consequent_5 = ($$anchor3) => {
        var ul = root_142();
        each(ul, 7, () => get(entry).postings, (posting, postingIndex) => posting.account + postingIndex, ($$anchor4, posting) => {
          var li_2 = root_152();
          var p_2 = child(li_2);
          var span_12 = sibling(child(p_2), 2);
          var text_12 = child(span_12, true);
          reset(span_12);
          var span_13 = sibling(span_12, 2);
          var a_1 = child(span_13);
          template_effect(() => set_attribute(a_1, "href", accountHref(get(posting).account)));
          var text_13 = child(a_1, true);
          reset(a_1);
          reset(span_13);
          var span_14 = sibling(span_13, 2);
          var text_14 = child(span_14, true);
          template_effect(() => set_text(text_14, amountText(get(posting).units)));
          reset(span_14);
          var span_15 = sibling(span_14, 2);
          var text_15 = child(span_15, true);
          template_effect(() => set_text(text_15, amountText(get(posting).cost)));
          reset(span_15);
          var span_16 = sibling(span_15, 2);
          var text_16 = child(span_16, true);
          template_effect(() => set_text(text_16, amountText(get(posting).price)));
          reset(span_16);
          reset(p_2);
          reset(li_2);
          template_effect(() => {
            set_text(text_12, get(posting).flag ?? "");
            set_text(text_13, get(posting).account);
          });
          append($$anchor4, li_2);
        });
        reset(ul);
        append($$anchor3, ul);
      };
      if_block(node_10, ($$render) => {
        if (get(entry).postings?.length) $$render(consequent_5);
      });
    }
    var node_11 = sibling(node_10, 2);
    {
      var consequent_6 = ($$anchor3) => {
        var dl = root_162();
        each(dl, 5, () => get(entry).metadata, (meta) => meta.key, ($$anchor4, meta) => {
          var fragment_5 = root_172();
          var dt = first_child(fragment_5);
          var text_17 = child(dt);
          reset(dt);
          var dd = sibling(dt, 2);
          var text_18 = child(dd, true);
          reset(dd);
          template_effect(() => {
            set_text(text_17, `${get(meta).key ?? ""}:`);
            set_text(text_18, get(meta).value);
          });
          append($$anchor4, fragment_5);
        });
        reset(dl);
        append($$anchor3, dl);
      };
      if_block(node_11, ($$render) => {
        if (get(entry).metadata?.length) $$render(consequent_6);
      });
    }
    reset(li_1);
    template_effect(() => {
      set_class(li_1, get(class_derived));
      toggle_class(li_1, "show-full-entry", get(class_directive_1));
      set_text(text_2, get(entry).date);
      set_text(text_3, get(entry).flag ?? "");
    });
    append($$anchor2, li_1);
  });
  reset(ol);
  template_effect(() => {
    set_class(ol, `${get(listClasses) ?? ""} svelte-3l7lkh`);
    set_text(text_1, runningBalances() ? "Balance" : "Price");
  });
  append($$anchor, fragment);
  pop();
}
delegate(["click", "keydown"]);

// src/fava/reports/AccountReport.svelte
var root_19 = template(`<section class="state-panel error-panel" role="alert"> </section>`);
var root_23 = template(`<div class="headerline"><h2> </h2><span class="muted">Account detail</span></div> <!> <!>`, 1);
function AccountReport($$anchor, $$props) {
  push($$props, false);
  let adapter = prop($$props, "adapter", 8);
  let account = prop($$props, "account", 8);
  let balance = mutable_state(null);
  let journal = mutable_state(null);
  let error = mutable_state("");
  async function load() {
    try {
      const [accountValue, journalValue] = await Promise.all([
        adapter().load("account", { account: account() }),
        adapter().load("journal", { account: account() })
      ]);
      set(balance, parseTableReport(accountValue));
      set(journal, parseTableReport(journalValue));
    } catch (value) {
      set(error, value instanceof Error ? value.message : "The account report could not be loaded.");
    }
  }
  load();
  init();
  var fragment = comment();
  var node = first_child(fragment);
  {
    var consequent = ($$anchor2) => {
      var section = root_19();
      var text2 = child(section, true);
      reset(section);
      template_effect(() => set_text(text2, get(error)));
      append($$anchor2, section);
    };
    var alternate = ($$anchor2) => {
      var fragment_1 = root_23();
      var div = first_child(fragment_1);
      var h2 = child(div);
      var text_1 = child(h2, true);
      reset(h2);
      next();
      reset(div);
      var node_1 = sibling(div, 2);
      {
        var consequent_1 = ($$anchor3) => {
          GenericReport($$anchor3, {
            get report() {
              return get(balance);
            },
            title: "Balance"
          });
        };
        if_block(node_1, ($$render) => {
          if (get(balance)) $$render(consequent_1);
        });
      }
      var node_2 = sibling(node_1, 2);
      {
        var consequent_2 = ($$anchor3) => {
          JournalReport($$anchor3, {
            get report() {
              return get(journal);
            },
            get account() {
              return account();
            }
          });
        };
        if_block(node_2, ($$render) => {
          if (get(journal)) $$render(consequent_2);
        });
      }
      template_effect(() => set_text(text_1, account()));
      append($$anchor2, fragment_1);
    };
    if_block(node, ($$render) => {
      if (get(error)) $$render(consequent);
      else $$render(alternate, false);
    });
  }
  append($$anchor, fragment);
  pop();
}

// src/fava/reports/ImportReport.svelte
var root_110 = template(`<option> </option>`);
var root_35 = template(`<li> </li>`);
var root_24 = template(`<ul class="diagnostics svelte-1iufbd5"></ul>`);
var root_54 = template(`<tr><td> </td><td> </td><td> </td><td> </td></tr>`);
var root_43 = template(`<table><thead><tr><th>Date</th><th>Account</th><th>Units</th><th>Currency</th></tr></thead><tbody></tbody></table>`);
var root5 = template(`<div class="headerline"><h2>Import</h2><span class="muted svelte-1iufbd5">Preview before commit</span></div> <div class="toolbar svelte-1iufbd5"><label>Source path <input></label> <label>Adapter <select><option>Beancount</option><option>CSV</option></select></label> <label>Target <select></select></label></div> <textarea class="import-buffer svelte-1iufbd5" placeholder="Paste Beancount or CSV content" spellcheck="false"></textarea> <div class="toolbar svelte-1iufbd5"><button type="button">Preview</button> <button type="button">Commit</button> <span class="muted svelte-1iufbd5" role="status"> </span></div> <!> <!>`, 1);
function ImportReport($$anchor, $$props) {
  push($$props, false);
  let adapter = prop($$props, "adapter", 8);
  let paths = mutable_state([]);
  let target2 = mutable_state("");
  let importPath = mutable_state("import.bean");
  let adapterID = mutable_state("beancount");
  let content = mutable_state("");
  let snapshotID = "";
  let previewID = mutable_state("");
  let valid = mutable_state(false);
  let diagnostics = mutable_state([]);
  let rows = mutable_state([]);
  let status = mutable_state("");
  async function loadTargets() {
    const value = await adapter().load("import");
    set(paths, value.paths ?? []);
    set(target2, value.entry || get(paths)[0] || "");
    snapshotID = value.snapshot_id ?? "";
  }
  async function request(path, body) {
    const response = await fetch(`/api/v1/import/${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    const value = await response.json();
    if (!response.ok) throw new Error(value.error || `Import request failed (${response.status})`);
    return value;
  }
  async function preview() {
    set(status, "Previewing\u2026");
    set(previewID, "");
    try {
      const value = await request("preview", {
        path: get(importPath),
        content: get(content),
        adapter: get(adapterID),
        mapping: {}
      });
      set(previewID, value.preview_id ?? "");
      set(valid, Boolean(value.valid));
      set(diagnostics, value.diagnostics ?? []);
      set(rows, value.rows?.rows ?? []);
      set(status, get(valid) ? `Preview ready: ${get(previewID)}` : "Preview has diagnostics");
    } catch (value) {
      set(status, value instanceof Error ? value.message : "Preview failed.");
    }
  }
  async function commit() {
    if (!get(previewID)) return;
    set(status, "Committing\u2026");
    try {
      const value = await request("commit", {
        preview_id: get(previewID),
        target: get(target2),
        expected_snapshot_id: snapshotID
      });
      snapshotID = value.snapshot_id ?? snapshotID;
      set(status, `Committed; backup: ${value.backup}`);
      set(previewID, "");
    } catch (value) {
      set(status, value instanceof Error ? value.message : "Commit failed.");
    }
  }
  onMount(() => {
    void loadTargets();
  });
  init();
  var fragment = root5();
  var div = sibling(first_child(fragment), 2);
  var label = child(div);
  var input = sibling(child(label));
  remove_input_defaults(input);
  reset(label);
  var label_1 = sibling(label, 2);
  var select = sibling(child(label_1));
  template_effect(() => {
    get(adapterID);
    invalidate_inner_signals(() => {
    });
  });
  var option = child(select);
  option.value = null == (option.__value = "beancount") ? "" : "beancount";
  var option_1 = sibling(option);
  option_1.value = null == (option_1.__value = "csv") ? "" : "csv";
  reset(select);
  reset(label_1);
  var label_2 = sibling(label_1, 2);
  var select_1 = sibling(child(label_2));
  template_effect(() => {
    get(target2);
    invalidate_inner_signals(() => {
      get(paths);
    });
  });
  each(select_1, 5, () => get(paths), (path) => path, ($$anchor2, path) => {
    var option_2 = root_110();
    var option_2_value = {};
    var text2 = child(option_2, true);
    reset(option_2);
    template_effect(() => {
      if (option_2_value !== (option_2_value = get(path))) {
        option_2.value = null == (option_2.__value = get(path)) ? "" : get(path);
      }
      set_text(text2, get(path));
    });
    append($$anchor2, option_2);
  });
  reset(select_1);
  reset(label_2);
  reset(div);
  var textarea = sibling(div, 2);
  remove_textarea_child(textarea);
  var div_1 = sibling(textarea, 2);
  var button = child(div_1);
  var button_1 = sibling(button, 2);
  var span = sibling(button_1, 2);
  var text_1 = child(span, true);
  reset(span);
  reset(div_1);
  var node = sibling(div_1, 2);
  {
    var consequent = ($$anchor2) => {
      var ul = root_24();
      each(ul, 5, () => get(diagnostics), (diagnostic) => diagnostic.code + diagnostic.line + diagnostic.message, ($$anchor3, diagnostic) => {
        var li = root_35();
        var text_2 = child(li);
        reset(li);
        template_effect(() => set_text(text_2, `${get(diagnostic).path ?? ""}:${get(diagnostic).line ?? ""}: ${get(diagnostic).message ?? ""}`));
        append($$anchor3, li);
      });
      reset(ul);
      append($$anchor2, ul);
    };
    if_block(node, ($$render) => {
      if (get(diagnostics).length) $$render(consequent);
    });
  }
  var node_1 = sibling(node, 2);
  {
    var consequent_1 = ($$anchor2) => {
      var table = root_43();
      var tbody = sibling(child(table));
      each(tbody, 5, () => get(rows), index, ($$anchor3, row) => {
        var tr = root_54();
        var td = child(tr);
        var text_3 = child(td, true);
        template_effect(() => set_text(text_3, String(get(row).date ?? "")));
        reset(td);
        var td_1 = sibling(td);
        var text_4 = child(td_1, true);
        template_effect(() => set_text(text_4, String(get(row).account ?? "")));
        reset(td_1);
        var td_2 = sibling(td_1);
        var text_5 = child(td_2, true);
        template_effect(() => set_text(text_5, String(get(row).units ?? "")));
        reset(td_2);
        var td_3 = sibling(td_2);
        var text_6 = child(td_3, true);
        template_effect(() => set_text(text_6, String(get(row).currency ?? "")));
        reset(td_3);
        reset(tr);
        append($$anchor3, tr);
      });
      reset(tbody);
      reset(table);
      append($$anchor2, table);
    };
    if_block(node_1, ($$render) => {
      if (get(rows).length) $$render(consequent_1);
    });
  }
  template_effect(() => {
    button_1.disabled = !get(previewID) || !get(valid);
    set_text(text_1, get(status));
  });
  bind_value(input, () => get(importPath), ($$value) => set(importPath, $$value));
  bind_select_value(select, () => get(adapterID), ($$value) => set(adapterID, $$value));
  bind_select_value(select_1, () => get(target2), ($$value) => set(target2, $$value));
  bind_value(textarea, () => get(content), ($$value) => set(content, $$value));
  event("click", button, preview);
  event("click", button_1, commit);
  append($$anchor, fragment);
  pop();
}

// src/fava/reports/QueryReport.svelte
var root_111 = template(`<p class="error-panel svelte-1k81p2c" role="alert"> </p>`);
var root_36 = template(`<p><a class="button">Export CSV</a></p> <!>`, 1);
var root6 = template(`<div class="headerline"><h2>Query</h2></div> <form class="query-form svelte-1k81p2c"><label for="query-editor">BeanQuery</label> <textarea id="query-editor" spellcheck="false" rows="4" class="svelte-1k81p2c"></textarea> <button type="submit"> </button></form> <!>`, 1);
function QueryReport($$anchor, $$props) {
  push($$props, false);
  let adapter = prop($$props, "adapter", 8);
  let queryText = mutable_state("SELECT account, balance FROM accounts ORDER BY account");
  let result = mutable_state(null);
  let loading = mutable_state(false);
  let error = mutable_state("");
  async function run2() {
    set(loading, true);
    set(error, "");
    try {
      set(result, parseTableReport(await adapter().load("query", { query_string: get(queryText) })));
    } catch (value) {
      set(error, value instanceof Error ? value.message : "The query could not be evaluated.");
    } finally {
      set(loading, false);
    }
  }
  onMount(() => {
    void run2();
  });
  init();
  var fragment = root6();
  var form = sibling(first_child(fragment), 2);
  var textarea = sibling(child(form), 2);
  remove_textarea_child(textarea);
  var button = sibling(textarea, 2);
  var text2 = child(button, true);
  reset(button);
  reset(form);
  var node = sibling(form, 2);
  {
    var consequent = ($$anchor2) => {
      var p = root_111();
      var text_1 = child(p, true);
      reset(p);
      template_effect(() => set_text(text_1, get(error)));
      append($$anchor2, p);
    };
    var alternate = ($$anchor2) => {
      var fragment_1 = comment();
      var node_1 = first_child(fragment_1);
      {
        var consequent_1 = ($$anchor3) => {
          var fragment_2 = root_36();
          var p_1 = first_child(fragment_2);
          var a = child(p_1);
          template_effect(() => set_attribute(a, "href", `/api/v1/query?q=${encodeURIComponent(get(queryText))}&format=csv`));
          reset(p_1);
          var node_2 = sibling(p_1, 2);
          GenericReport(node_2, {
            get report() {
              return get(result);
            },
            title: "Query result"
          });
          append($$anchor3, fragment_2);
        };
        if_block(
          node_1,
          ($$render) => {
            if (get(result)) $$render(consequent_1);
          },
          true
        );
      }
      append($$anchor2, fragment_1);
    };
    if_block(node, ($$render) => {
      if (get(error)) $$render(consequent);
      else $$render(alternate, false);
    });
  }
  template_effect(() => {
    button.disabled = get(loading);
    set_text(text2, get(loading) ? "Running\u2026" : "Run query");
  });
  bind_value(textarea, () => get(queryText), ($$value) => set(queryText, $$value));
  event("submit", form, preventDefault(run2));
  append($$anchor, fragment);
  pop();
}

// src/fava/reports/EditorReport.svelte
var root_113 = template(`<option> </option>`);
var root_37 = template(`<li> </li>`);
var root_25 = template(`<ul class="diagnostics svelte-1c22f1y"></ul>`);
var root7 = template(`<div class="headerline"><h2>Editor</h2><span class="muted svelte-1c22f1y">Reviewed writes only</span></div> <div class="editor-layout svelte-1c22f1y"><aside class="editor-files svelte-1c22f1y"><label for="editor-file">Files</label> <select id="editor-file" class="svelte-1c22f1y"></select></aside> <section class="editor-pane svelte-1c22f1y"><div class="toolbar svelte-1c22f1y"><button id="editor-validate" type="button">Validate</button> <button id="editor-save" type="button">Save</button> <span class="muted svelte-1c22f1y" role="status"> </span></div> <textarea id="editor-buffer" spellcheck="false" aria-label="Ledger source" class="svelte-1c22f1y"></textarea> <!></section></div>`, 1);
function EditorReport($$anchor, $$props) {
  push($$props, false);
  let adapter = prop($$props, "adapter", 8);
  let paths = mutable_state([]);
  let selected = mutable_state("");
  let content = mutable_state("");
  let snapshotID = "";
  let status = mutable_state("");
  let diagnostics = mutable_state([]);
  let loading = mutable_state(true);
  async function loadIndex() {
    const value = await adapter().load("editor");
    set(paths, value.paths ?? []);
    snapshotID = value.snapshot_id ?? "";
    set(selected, value.entry || get(paths)[0] || "");
    if (get(selected)) await loadFile();
  }
  async function loadFile() {
    if (!get(selected)) return;
    set(loading, true);
    try {
      const value = await adapter().load("editor", { path: get(selected) });
      set(content, value.content);
      snapshotID = value.snapshot_id;
      set(diagnostics, []);
      set(status, "");
    } catch (value) {
      set(status, value instanceof Error ? value.message : "Unable to load source file.");
    } finally {
      set(loading, false);
    }
  }
  async function send(path, body) {
    const response = await fetch(`/api/v1/editor/${path}`, {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body)
    });
    const value = await response.json();
    if (!response.ok) throw new Error(value.error || `Editor request failed (${response.status})`);
    return value;
  }
  async function validate() {
    set(status, "Validating\u2026");
    try {
      const value = await send("validate", {
        path: get(selected),
        content: get(content),
        expected_snapshot_id: snapshotID
      });
      set(diagnostics, value.diagnostics ?? []);
      set(status, value.valid ? "Valid" : "Diagnostics found");
    } catch (value) {
      set(status, value instanceof Error ? value.message : "Validation failed.");
    }
  }
  async function save() {
    set(status, "Saving\u2026");
    try {
      const value = await send("save", {
        path: get(selected),
        content: get(content),
        expected_snapshot_id: snapshotID
      });
      set(diagnostics, value.diagnostics ?? []);
      if (value.published) {
        snapshotID = value.snapshot_id;
        set(status, `Saved; backup: ${value.backup}`);
      } else {
        set(status, "Save rejected; the previous snapshot remains active.");
      }
    } catch (value) {
      set(status, value instanceof Error ? value.message : "Save failed.");
    }
  }
  onMount(() => {
    void loadIndex();
  });
  init();
  var fragment = root7();
  var div = sibling(first_child(fragment), 2);
  var aside = child(div);
  var select = sibling(child(aside), 2);
  template_effect(() => {
    get(selected);
    invalidate_inner_signals(() => {
      Math;
      get(paths);
      loadFile;
    });
  });
  template_effect(() => set_attribute(select, "size", Math.min(Math.max(get(paths).length, 2), 12)));
  each(select, 5, () => get(paths), (path) => path, ($$anchor2, path) => {
    var option = root_113();
    var option_value = {};
    var text2 = child(option, true);
    reset(option);
    template_effect(() => {
      if (option_value !== (option_value = get(path))) {
        option.value = null == (option.__value = get(path)) ? "" : get(path);
      }
      set_text(text2, get(path));
    });
    append($$anchor2, option);
  });
  reset(select);
  reset(aside);
  var section = sibling(aside, 2);
  var div_1 = child(section);
  var button = child(div_1);
  var button_1 = sibling(button, 2);
  var span = sibling(button_1, 2);
  var text_1 = child(span, true);
  reset(span);
  reset(div_1);
  var textarea = sibling(div_1, 2);
  remove_textarea_child(textarea);
  var node = sibling(textarea, 2);
  {
    var consequent = ($$anchor2) => {
      var ul = root_25();
      each(ul, 5, () => get(diagnostics), (diagnostic) => diagnostic.code + diagnostic.line + diagnostic.message, ($$anchor3, diagnostic) => {
        var li = root_37();
        var text_2 = child(li);
        reset(li);
        template_effect(() => set_text(text_2, `${get(diagnostic).path ?? ""}:${get(diagnostic).line ?? ""}: ${get(diagnostic).message ?? ""}`));
        append($$anchor3, li);
      });
      reset(ul);
      append($$anchor2, ul);
    };
    if_block(node, ($$render) => {
      if (get(diagnostics).length) $$render(consequent);
    });
  }
  reset(section);
  reset(div);
  template_effect(() => {
    button.disabled = get(loading);
    button_1.disabled = get(loading);
    set_text(text_1, get(status));
  });
  bind_select_value(select, () => get(selected), ($$value) => set(selected, $$value));
  event("change", select, loadFile);
  event("click", button, validate);
  event("click", button_1, save);
  bind_value(textarea, () => get(content), ($$value) => set(content, $$value));
  append($$anchor, fragment);
  pop();
}

// src/fava/tree-table/TreeTableNode.svelte
var on_click2 = (_, onToggle, node) => onToggle()(node().account);
var root_114 = template(`<button type="button" class="unset expander svelte-xjp7mv"> </button>`);
var root_26 = template(`<span class="num svelte-xjp7mv"> </span>`);
var root_44 = template(`<span class="other-line svelte-xjp7mv"> </span>`);
var root_38 = template(`<span class="other num svelte-xjp7mv"></span>`);
var root_55 = template(`<ol></ol>`);
var root8 = template(`<li><p><span class="account-cell svelte-xjp7mv"><!> <a class="account svelte-xjp7mv"> </a></span> <!> <!></p> <!></li>`);
function TreeTableNode($$anchor, $$props) {
  push($$props, false);
  const isCollapsed = mutable_state();
  const shown = mutable_state();
  const hasChildren = mutable_state();
  const leaf = mutable_state();
  const otherAmounts = mutable_state();
  let node = prop($$props, "node", 8);
  let currencies = prop($$props, "currencies", 24, () => []);
  let otherCurrencies = prop($$props, "otherCurrencies", 24, () => []);
  let renderCommas = prop($$props, "renderCommas", 8, false);
  let collapsed = prop($$props, "collapsed", 8);
  let depth = prop($$props, "depth", 8, 0);
  let onToggle = prop($$props, "onToggle", 8);
  legacy_pre_effect(
    () => (deep_read_state(collapsed()), deep_read_state(node())),
    () => {
      set(isCollapsed, collapsed().has(node().account));
    }
  );
  legacy_pre_effect(
    () => (get(isCollapsed), deep_read_state(node())),
    () => {
      set(shown, get(isCollapsed) || !node().has_txns ? node().balance_children : node().balance);
    }
  );
  legacy_pre_effect(() => deep_read_state(node()), () => {
    set(hasChildren, node().children.length > 0);
  });
  legacy_pre_effect(() => deep_read_state(node()), () => {
    set(leaf, node().account.includes(":") ? node().account.slice(node().account.lastIndexOf(":") + 1) : node().account);
  });
  legacy_pre_effect(
    () => (deep_read_state(otherCurrencies()), get(shown), formatAmount, deep_read_state(renderCommas())),
    () => {
      set(otherAmounts, otherCurrencies().filter((currency) => get(shown)[currency]).map((currency) => `${formatAmount(get(shown)[currency], renderCommas())} ${currency}`));
    }
  );
  legacy_pre_effect_reset();
  init();
  var li = root8();
  var p = child(li);
  var span = child(p);
  var node_1 = child(span);
  {
    var consequent = ($$anchor2) => {
      var button = root_114();
      button.__click = [on_click2, onToggle, node];
      var text2 = child(button, true);
      reset(button);
      template_effect(() => {
        set_attribute(button, "aria-label", get(isCollapsed) ? `Expand ${node().account}` : `Collapse ${node().account}`);
        set_attribute(button, "aria-expanded", !get(isCollapsed));
        set_text(text2, get(isCollapsed) ? "\u25B8" : "\u25BE");
      });
      append($$anchor2, button);
    };
    if_block(node_1, ($$render) => {
      if (get(hasChildren)) $$render(consequent);
    });
  }
  var a = sibling(node_1, 2);
  template_effect(() => set_attribute(a, "href", `/account/${encodeURIComponent(node().account)}`));
  var text_1 = child(a, true);
  reset(a);
  reset(span);
  var node_2 = sibling(span, 2);
  each(node_2, 1, currencies, (currency) => currency, ($$anchor2, currency) => {
    var span_1 = root_26();
    var text_2 = child(span_1, true);
    template_effect(() => set_text(text_2, formatAmount(get(shown)[get(currency)], renderCommas())));
    reset(span_1);
    template_effect(() => {
      set_attribute(span_1, "title", get(shown)[get(currency)]?.exact ?? "");
      toggle_class(span_1, "dimmed", !get(shown)[get(currency)]);
    });
    append($$anchor2, span_1);
  });
  var node_3 = sibling(node_2, 2);
  {
    var consequent_1 = ($$anchor2) => {
      var span_2 = root_38();
      each(span_2, 5, () => get(otherAmounts), (amount) => amount, ($$anchor3, amount) => {
        var span_3 = root_44();
        var text_3 = child(span_3, true);
        reset(span_3);
        template_effect(() => set_text(text_3, get(amount)));
        append($$anchor3, span_3);
      });
      reset(span_2);
      append($$anchor2, span_2);
    };
    if_block(node_3, ($$render) => {
      if (otherCurrencies().length) $$render(consequent_1);
    });
  }
  reset(p);
  var node_4 = sibling(p, 2);
  {
    var consequent_2 = ($$anchor2) => {
      var ol = root_55();
      each(ol, 5, () => node().children, (child2) => child2.account, ($$anchor3, child2) => {
        var fragment = comment();
        var node_5 = first_child(fragment);
        var depth_1 = derived_safe_equal(() => depth() + 1);
        TreeTableNode(node_5, {
          get node() {
            return get(child2);
          },
          get currencies() {
            return currencies();
          },
          get otherCurrencies() {
            return otherCurrencies();
          },
          get renderCommas() {
            return renderCommas();
          },
          get collapsed() {
            return collapsed();
          },
          get depth() {
            return get(depth_1);
          },
          get onToggle() {
            return onToggle();
          }
        });
        append($$anchor3, fragment);
      });
      reset(ol);
      append($$anchor2, ol);
    };
    if_block(node_4, ($$render) => {
      if (!get(isCollapsed) && get(hasChildren)) $$render(consequent_2);
    });
  }
  reset(li);
  template_effect(() => {
    set_attribute(span, "style", `--account-indent: ${depth()}em`);
    set_text(text_1, get(leaf));
  });
  append($$anchor, li);
  pop();
}
delegate(["click"]);

// src/fava/tree-table/TreeTable.svelte
var root_115 = template(`<span class="num"> </span>`);
var root_27 = template(`<span class="other">Other</span>`);
var root9 = template(`<ol class="flex-table tree-table-new" data-tree-table=""><li class="head"><p><span class="account-cell svelte-1xyup19"> </span> <!> <!></p></li> <!></ol>`);
function TreeTable($$anchor, $$props) {
  push($$props, false);
  const present = mutable_state();
  const columns = mutable_state();
  const other = mutable_state();
  const roots = mutable_state();
  let tree = prop($$props, "tree", 8);
  let end = prop($$props, "end", 8, null);
  let operatingCurrencies = prop($$props, "operatingCurrencies", 24, () => []);
  let renderCommas = prop($$props, "renderCommas", 8, false);
  let collapsed = mutable_state(/* @__PURE__ */ new Set());
  function toggle(account) {
    const next2 = new Set(get(collapsed));
    if (next2.has(account)) next2.delete(account);
    else next2.add(account);
    set(collapsed, next2);
  }
  legacy_pre_effect(
    () => (currenciesInTree, deep_read_state(tree())),
    () => {
      set(present, currenciesInTree(tree()));
    }
  );
  legacy_pre_effect(
    () => (deep_read_state(operatingCurrencies()), get(present)),
    () => {
      set(columns, operatingCurrencies().filter((currency) => get(present).includes(currency)));
    }
  );
  legacy_pre_effect(() => (get(present), get(columns)), () => {
    set(other, get(present).filter((currency) => !get(columns).includes(currency)));
  });
  legacy_pre_effect(() => deep_read_state(tree()), () => {
    set(roots, tree().account === "" ? tree().children : [tree()]);
  });
  legacy_pre_effect_reset();
  init();
  var ol = root9();
  var li = child(ol);
  var p = child(li);
  var span = child(p);
  var text2 = child(span, true);
  reset(span);
  var node_1 = sibling(span, 2);
  each(node_1, 1, () => get(columns), (currency) => currency, ($$anchor2, currency) => {
    var span_1 = root_115();
    var text_1 = child(span_1, true);
    reset(span_1);
    template_effect(() => {
      set_attribute(span_1, "title", get(currency));
      set_text(text_1, get(currency));
    });
    append($$anchor2, span_1);
  });
  var node_2 = sibling(node_1, 2);
  {
    var consequent = ($$anchor2) => {
      var span_2 = root_27();
      append($$anchor2, span_2);
    };
    if_block(node_2, ($$render) => {
      if (get(other).length) $$render(consequent);
    });
  }
  reset(p);
  reset(li);
  var node_3 = sibling(li, 2);
  each(node_3, 1, () => get(roots), (node) => node.account, ($$anchor2, node) => {
    TreeTableNode($$anchor2, {
      get node() {
        return get(node);
      },
      get currencies() {
        return get(columns);
      },
      get otherCurrencies() {
        return get(other);
      },
      get renderCommas() {
        return renderCommas();
      },
      get collapsed() {
        return get(collapsed);
      },
      onToggle: toggle
    });
  });
  reset(ol);
  template_effect(() => {
    set_attribute(ol, "data-end", end() ?? void 0);
    set_text(text2, tree().account || "Accounts");
  });
  append($$anchor, ol);
  pop();
}

// src/fava/reports/TreeReport.svelte
var root_39 = template(`<button type="button" class="unset svelte-1dzjfb9"> </button>`);
var root_28 = template(`<nav class="chart-picker svelte-1dzjfb9"></nav>`);
var root_116 = template(`<div class="report-charts"><!> <!></div>`);
var root10 = template(`<!> <div class="row"><div class="column"></div> <div class="column"></div></div>`, 1);
function TreeReport($$anchor, $$props) {
  push($$props, false);
  const chart2 = mutable_state();
  const firstColumn = mutable_state();
  const secondColumn = mutable_state();
  const end = mutable_state();
  let report = prop($$props, "report", 8);
  let locale = prop($$props, "locale", 8, "en");
  let operatingCurrencies = prop($$props, "operatingCurrencies", 24, () => []);
  let renderCommas = prop($$props, "renderCommas", 8, false);
  let selected = mutable_state(0);
  legacy_pre_effect(
    () => (get(selected), deep_read_state(report())),
    () => {
      if (get(selected) >= report().charts.length) set(selected, 0);
    }
  );
  legacy_pre_effect(
    () => (deep_read_state(report()), get(selected)),
    () => {
      set(chart2, report().charts[get(selected)]);
    }
  );
  legacy_pre_effect(() => deep_read_state(report()), () => {
    set(firstColumn, report().trees.slice(0, 2));
  });
  legacy_pre_effect(() => deep_read_state(report()), () => {
    set(secondColumn, report().trees.slice(2));
  });
  legacy_pre_effect(() => deep_read_state(report()), () => {
    set(end, report().date_range?.end ?? null);
  });
  legacy_pre_effect_reset();
  init();
  var fragment = root10();
  var node = first_child(fragment);
  {
    var consequent_1 = ($$anchor2) => {
      var div = root_116();
      var node_1 = child(div);
      ReportChart(node_1, {
        get chart() {
          return get(chart2);
        },
        get locale() {
          return locale();
        }
      });
      var node_2 = sibling(node_1, 2);
      {
        var consequent = ($$anchor3) => {
          var nav = root_28();
          each(nav, 7, () => report().charts, (option) => option.title, ($$anchor4, option, index2) => {
            var button = root_39();
            button.__click = () => set(selected, get(index2));
            var text2 = child(button, true);
            reset(button);
            template_effect(() => {
              set_attribute(button, "aria-pressed", get(index2) === get(selected));
              toggle_class(button, "selected", get(index2) === get(selected));
              set_text(text2, get(option).title);
            });
            append($$anchor4, button);
          });
          reset(nav);
          template_effect(() => set_attribute(nav, "aria-label", get(chart2).title));
          append($$anchor3, nav);
        };
        if_block(node_2, ($$render) => {
          if (report().charts.length > 1) $$render(consequent);
        });
      }
      reset(div);
      append($$anchor2, div);
    };
    if_block(node, ($$render) => {
      if (get(chart2)) $$render(consequent_1);
    });
  }
  var div_1 = sibling(node, 2);
  var div_2 = child(div_1);
  each(div_2, 5, () => get(firstColumn), (tree) => tree.account, ($$anchor2, tree) => {
    TreeTable($$anchor2, {
      get tree() {
        return get(tree);
      },
      get end() {
        return get(end);
      },
      get operatingCurrencies() {
        return operatingCurrencies();
      },
      get renderCommas() {
        return renderCommas();
      }
    });
  });
  reset(div_2);
  var div_3 = sibling(div_2, 2);
  each(div_3, 5, () => get(secondColumn), (tree) => tree.account, ($$anchor2, tree) => {
    TreeTable($$anchor2, {
      get tree() {
        return get(tree);
      },
      get end() {
        return get(end);
      },
      get operatingCurrencies() {
        return operatingCurrencies();
      },
      get renderCommas() {
        return renderCommas();
      }
    });
  });
  reset(div_3);
  reset(div_1);
  append($$anchor, fragment);
  pop();
}
delegate(["click"]);

// src/fava/reports/UtilityReport.svelte
var root_117 = template(`<section class="state-panel" role="status">Loading\u2026</section>`);
var root_310 = template(`<section class="state-panel error-panel" role="alert"> </section>`);
var root_63 = template(`<details open class="svelte-16ywk3n"><summary> </summary> <div> </div></details>`);
var root_56 = template(`<div class="headerline"><h2>Help</h2></div> <!>`, 1);
var root_84 = template(`<div class="headerline"><h2> </h2></div> <pre class="source-content svelte-16ywk3n"> </pre>`, 1);
var root_118 = template(`<li><a> </a></li>`);
var root_104 = template(`<div class="headerline"><h2>Source files</h2></div> <ul class="source-list svelte-16ywk3n"></ul>`, 1);
var root_143 = template(`<tr><th scope="row"> </th><td> </td></tr>`);
var root_133 = template(`<div class="headerline"><h2>Options</h2></div> <h3>Color scheme</h3> <div class="color-scheme svelte-16ywk3n" role="radiogroup" aria-label="Color scheme"><label><input type="radio" name="color-scheme" value="system"> \u2699\uFE0F System</label> <label><input type="radio" name="color-scheme" value="dark"> \u{1F319} Dark</label> <label><input type="radio" name="color-scheme" value="light"> \u2600\uFE0F Light</label></div> <h3>Fava options</h3> <table class="svelte-16ywk3n"><thead><tr><th>Key</th><th>Value</th></tr></thead><tbody><tr><th scope="row">locale</th><td><select id="fava-option-locale"><option>English</option><option>\u7B80\u4F53\u4E2D\u6587</option></select></td></tr><tr><th scope="row">theme</th><td> </td></tr></tbody></table> <h3>Beancount options</h3> <table class="svelte-16ywk3n"><thead><tr><th>Option</th><th>Value</th></tr></thead><tbody></tbody></table>`, 1);
var root_153 = template(`<div class="headerline"><h2> </h2></div> <pre class="svelte-16ywk3n"> </pre>`, 1);
function UtilityReport($$anchor, $$props) {
  push($$props, false);
  let adapter = prop($$props, "adapter", 8);
  let route = prop($$props, "route", 8);
  let query = prop($$props, "query", 24, () => ({}));
  let locale = prop($$props, "locale", 8, "en");
  let theme = prop($$props, "theme", 8, "system");
  let onLocale = prop($$props, "onLocale", 8, () => {
  });
  let onTheme = prop($$props, "onTheme", 8, () => {
  });
  let loading = mutable_state(true);
  let error = mutable_state("");
  let data = mutable_state(null);
  async function load() {
    set(loading, true);
    set(error, "");
    try {
      set(data, await adapter().load(route(), query()));
    } catch (value) {
      set(error, value instanceof Error ? value.message : "The page could not be loaded.");
    } finally {
      set(loading, false);
    }
  }
  load();
  function objectEntries(value) {
    return value && typeof value === "object" && !Array.isArray(value) ? Object.entries(value) : [];
  }
  init();
  var fragment = comment();
  var node = first_child(fragment);
  {
    var consequent = ($$anchor2) => {
      var section_1 = root_117();
      append($$anchor2, section_1);
    };
    var alternate_5 = ($$anchor2) => {
      var fragment_1 = comment();
      var node_1 = first_child(fragment_1);
      {
        var consequent_1 = ($$anchor3) => {
          var section_2 = root_310();
          var text2 = child(section_2, true);
          reset(section_2);
          template_effect(() => set_text(text2, get(error)));
          append($$anchor3, section_2);
        };
        var alternate_4 = ($$anchor3) => {
          var fragment_2 = comment();
          var node_2 = first_child(fragment_2);
          {
            var consequent_2 = ($$anchor4) => {
              var fragment_3 = root_56();
              var node_3 = sibling(first_child(fragment_3), 2);
              each(node_3, 1, () => get(data).sections, (section) => section.id, ($$anchor5, section) => {
                var details = root_63();
                var summary = child(details);
                var text_1 = child(summary, true);
                reset(summary);
                var div = sibling(summary, 2);
                var text_2 = child(div, true);
                reset(div);
                reset(details);
                template_effect(() => {
                  set_text(text_1, get(section).title);
                  set_text(text_2, get(section).body);
                });
                append($$anchor5, details);
              });
              append($$anchor4, fragment_3);
            };
            var alternate_3 = ($$anchor4) => {
              var fragment_4 = comment();
              var node_4 = first_child(fragment_4);
              {
                var consequent_3 = ($$anchor5) => {
                  var fragment_5 = root_84();
                  var div_1 = first_child(fragment_5);
                  var h2 = child(div_1);
                  var text_3 = child(h2, true);
                  reset(h2);
                  reset(div_1);
                  var pre = sibling(div_1, 2);
                  var text_4 = child(pre, true);
                  reset(pre);
                  template_effect(() => {
                    set_text(text_3, get(data).path);
                    set_text(text_4, get(data).content);
                  });
                  append($$anchor5, fragment_5);
                };
                var alternate_2 = ($$anchor5) => {
                  var fragment_6 = comment();
                  var node_5 = first_child(fragment_6);
                  {
                    var consequent_4 = ($$anchor6) => {
                      var fragment_7 = root_104();
                      var ul = sibling(first_child(fragment_7), 2);
                      each(ul, 5, () => get(data).paths, (path) => path, ($$anchor7, path) => {
                        var li = root_118();
                        var a = child(li);
                        template_effect(() => set_attribute(a, "href", `/source?path=${encodeURIComponent(get(path))}`));
                        var text_5 = child(a, true);
                        reset(a);
                        reset(li);
                        template_effect(() => set_text(text_5, get(path)));
                        append($$anchor7, li);
                      });
                      reset(ul);
                      append($$anchor6, fragment_7);
                    };
                    var alternate_1 = ($$anchor6) => {
                      var fragment_8 = comment();
                      var node_6 = first_child(fragment_8);
                      {
                        var consequent_5 = ($$anchor7) => {
                          var fragment_9 = root_133();
                          var div_2 = sibling(first_child(fragment_9), 4);
                          var label = child(div_2);
                          var input = child(label);
                          remove_input_defaults(input);
                          next();
                          reset(label);
                          var label_1 = sibling(label, 2);
                          var input_1 = child(label_1);
                          remove_input_defaults(input_1);
                          next();
                          reset(label_1);
                          var label_2 = sibling(label_1, 2);
                          var input_2 = child(label_2);
                          remove_input_defaults(input_2);
                          next();
                          reset(label_2);
                          reset(div_2);
                          var table = sibling(div_2, 4);
                          var tbody = sibling(child(table));
                          var tr = child(tbody);
                          var td = sibling(child(tr));
                          var select = child(td);
                          init_select(select, locale);
                          var select_value;
                          var option = child(select);
                          option.value = null == (option.__value = "en") ? "" : "en";
                          var option_1 = sibling(option);
                          option_1.value = null == (option_1.__value = "zh-CN") ? "" : "zh-CN";
                          reset(select);
                          reset(td);
                          reset(tr);
                          var tr_1 = sibling(tr);
                          var td_1 = sibling(child(tr_1));
                          var text_6 = child(td_1, true);
                          reset(td_1);
                          reset(tr_1);
                          reset(tbody);
                          reset(table);
                          var table_1 = sibling(table, 4);
                          var tbody_1 = sibling(child(table_1));
                          each(tbody_1, 5, () => objectEntries(get(data)?.options), ([key, value]) => key, ($$anchor8, $$item) => {
                            let key = () => get($$item)[0];
                            let value = () => get($$item)[1];
                            var tr_2 = root_143();
                            var th = child(tr_2);
                            var text_7 = child(th, true);
                            reset(th);
                            var td_2 = sibling(th);
                            var text_8 = child(td_2, true);
                            template_effect(() => set_text(text_8, String(value())));
                            reset(td_2);
                            reset(tr_2);
                            template_effect(() => set_text(text_7, key()));
                            append($$anchor8, tr_2);
                          });
                          reset(tbody_1);
                          reset(table_1);
                          template_effect(() => {
                            set_checked(input, theme() === "system");
                            set_checked(input_1, theme() === "dark");
                            set_checked(input_2, theme() === "light");
                            if (select_value !== (select_value = locale())) {
                              select.value = null == (select.__value = locale()) ? "" : locale(), select_option(select, locale());
                            }
                            set_text(text_6, theme());
                          });
                          event("change", input, () => onTheme()("system"));
                          event("change", input_1, () => onTheme()("dark"));
                          event("change", input_2, () => onTheme()("light"));
                          event("change", select, (event2) => onLocale()(event2.currentTarget.value));
                          append($$anchor7, fragment_9);
                        };
                        var alternate = ($$anchor7) => {
                          var fragment_10 = root_153();
                          var div_3 = first_child(fragment_10);
                          var h2_1 = child(div_3);
                          var text_9 = child(h2_1, true);
                          reset(h2_1);
                          reset(div_3);
                          var pre_1 = sibling(div_3, 2);
                          var text_10 = child(pre_1, true);
                          template_effect(() => set_text(text_10, JSON.stringify(get(data), null, 2)));
                          reset(pre_1);
                          template_effect(() => set_text(text_9, route()));
                          append($$anchor7, fragment_10);
                        };
                        if_block(
                          node_6,
                          ($$render) => {
                            if (route() === "options") $$render(consequent_5);
                            else $$render(alternate, false);
                          },
                          true
                        );
                      }
                      append($$anchor6, fragment_8);
                    };
                    if_block(
                      node_5,
                      ($$render) => {
                        if (route() === "source" && get(data)?.paths) $$render(consequent_4);
                        else $$render(alternate_1, false);
                      },
                      true
                    );
                  }
                  append($$anchor5, fragment_6);
                };
                if_block(
                  node_4,
                  ($$render) => {
                    if (route() === "source" && get(data)?.content !== void 0) $$render(consequent_3);
                    else $$render(alternate_2, false);
                  },
                  true
                );
              }
              append($$anchor4, fragment_4);
            };
            if_block(
              node_2,
              ($$render) => {
                if (route() === "help" && get(data)?.sections) $$render(consequent_2);
                else $$render(alternate_3, false);
              },
              true
            );
          }
          append($$anchor3, fragment_2);
        };
        if_block(
          node_1,
          ($$render) => {
            if (get(error)) $$render(consequent_1);
            else $$render(alternate_4, false);
          },
          true
        );
      }
      append($$anchor2, fragment_1);
    };
    if_block(node, ($$render) => {
      if (get(loading)) $$render(consequent);
      else $$render(alternate_5, false);
    });
  }
  append($$anchor, fragment);
  pop();
}

// src/fava/lib/errors.ts
function errorWithCauses(error) {
  const msg = error.message;
  return error.cause instanceof Error ? `${msg}
  Caused by: ${errorWithCauses(error.cause)}` : error.message;
}

// src/fava/notifications.ts
var notificationList = /* @__PURE__ */ (() => {
  let value = null;
  return () => {
    if (value == null) {
      value = document.createElement("div");
      value.className = "notifications";
      value.style.right = "10px";
      document.body.appendChild(value);
    }
    const headerHeight = document.querySelector("header")?.getBoundingClientRect().height ?? 50;
    value.style.top = `${(headerHeight + 10).toString()}px`;
    return value;
  };
})();
function notify(msg, cls = "info", callback) {
  const notification = document.createElement("li");
  notification.classList.add(cls);
  notification.appendChild(document.createTextNode(msg));
  notificationList().append(notification);
  notification.addEventListener("click", () => {
    notification.remove();
    callback?.();
  });
  setTimeout(() => {
    notification.remove();
  }, 5e3);
}
function notify_err(error, msg = errorWithCauses) {
  if (error instanceof Error) {
    notify(msg(error), "error");
  }
  console.error(error);
}

// src/fava/components/ReportOutlet.svelte
var root_119 = template(`<section class="state-panel" role="status" aria-live="polite">Loading report\u2026</section>`);
var root_311 = template(`<section class="state-panel error-panel" role="alert"> </section>`);
var root_20 = template(`<section class="route-placeholder"><p class="headerline"><strong>Fava-aligned shell</strong></p> <h2> </h2> <p>This route is staged until its OrangeCount adapter contract is implemented.</p></section>`);
function ReportOutlet($$anchor, $$props) {
  push($$props, false);
  const requestKey = mutable_state();
  let adapter = prop($$props, "adapter", 8);
  let route = prop($$props, "route", 8);
  let query = prop($$props, "query", 24, () => ({}));
  let locale = prop($$props, "locale", 8, "en");
  let theme = prop($$props, "theme", 8, "system");
  let operatingCurrencies = prop($$props, "operatingCurrencies", 24, () => []);
  let renderCommas = prop($$props, "renderCommas", 8, false);
  let onLocale = prop($$props, "onLocale", 8, () => {
  });
  let onTheme = prop($$props, "onTheme", 8, () => {
  });
  let loadedKey = mutable_state("");
  let loading = mutable_state(false);
  let error = mutable_state(null);
  let report = mutable_state(null);
  let table = mutable_state(null);
  let journal = mutable_state(null);
  async function load(key) {
    set(loading, true);
    set(error, null);
    set(report, null);
    set(table, null);
    set(journal, null);
    if ([
      "query",
      "options",
      "help",
      "diagnostics",
      "source",
      "editor",
      "import"
    ].includes(route()) || ![
      "income_statement",
      "balance_sheet",
      "trial_balance",
      "accounts",
      "account",
      "journal",
      "holdings",
      "holdings_by_account",
      "holdings_by_currency",
      "holdings_by_root_account",
      "holdings_by_commodity",
      "commodities",
      "events",
      "documents",
      "statistics",
      "errors"
    ].includes(route())) {
      set(loading, false);
      return;
    }
    try {
      const payload = await adapter().load(route(), query());
      if (key !== get(requestKey)) return;
      if ([
        "income_statement",
        "balance_sheet",
        "trial_balance"
      ].includes(route())) {
        set(report, parseTreeReport(payload));
      } else if (route() === "journal") {
        set(journal, parseJournalReport(payload));
      } else {
        set(table, parseTableReport(payload));
      }
    } catch (value) {
      if (key !== get(requestKey)) return;
      notify_err(value);
      set(error, value instanceof Error ? value.message : "The report could not be loaded.");
    } finally {
      if (key === get(requestKey)) set(loading, false);
    }
  }
  legacy_pre_effect(
    () => (deep_read_state(route()), deep_read_state(query())),
    () => {
      set(requestKey, `${route()}?${new URLSearchParams(query()).toString()}`);
    }
  );
  legacy_pre_effect(() => (get(requestKey), get(loadedKey)), () => {
    if (get(requestKey) !== get(loadedKey)) {
      set(loadedKey, get(requestKey));
      void load(get(requestKey));
    }
  });
  legacy_pre_effect_reset();
  init();
  var fragment = comment();
  var node = first_child(fragment);
  {
    var consequent = ($$anchor2) => {
      var section = root_119();
      append($$anchor2, section);
    };
    var alternate_9 = ($$anchor2) => {
      var fragment_1 = comment();
      var node_1 = first_child(fragment_1);
      {
        var consequent_1 = ($$anchor3) => {
          var section_1 = root_311();
          var text2 = child(section_1, true);
          reset(section_1);
          template_effect(() => set_text(text2, get(error)));
          append($$anchor3, section_1);
        };
        var alternate_8 = ($$anchor3) => {
          var fragment_2 = comment();
          var node_2 = first_child(fragment_2);
          {
            var consequent_2 = ($$anchor4) => {
              QueryReport($$anchor4, {
                get adapter() {
                  return adapter();
                }
              });
            };
            var alternate_7 = ($$anchor4) => {
              var fragment_4 = comment();
              var node_3 = first_child(fragment_4);
              {
                var consequent_3 = ($$anchor5) => {
                  var account = derived_safe_equal(() => query().account || "");
                  AccountReport($$anchor5, {
                    get adapter() {
                      return adapter();
                    },
                    get account() {
                      return get(account);
                    }
                  });
                };
                var alternate_6 = ($$anchor5) => {
                  var fragment_6 = comment();
                  var node_4 = first_child(fragment_6);
                  {
                    var consequent_4 = ($$anchor6) => {
                      EditorReport($$anchor6, {
                        get adapter() {
                          return adapter();
                        }
                      });
                    };
                    var alternate_5 = ($$anchor6) => {
                      var fragment_8 = comment();
                      var node_5 = first_child(fragment_8);
                      {
                        var consequent_5 = ($$anchor7) => {
                          ImportReport($$anchor7, {
                            get adapter() {
                              return adapter();
                            }
                          });
                        };
                        var alternate_4 = ($$anchor7) => {
                          var fragment_10 = comment();
                          var node_6 = first_child(fragment_10);
                          {
                            var consequent_6 = ($$anchor8) => {
                              UtilityReport($$anchor8, {
                                get adapter() {
                                  return adapter();
                                },
                                get route() {
                                  return route();
                                },
                                get query() {
                                  return query();
                                },
                                get locale() {
                                  return locale();
                                },
                                get theme() {
                                  return theme();
                                },
                                get onLocale() {
                                  return onLocale();
                                },
                                get onTheme() {
                                  return onTheme();
                                }
                              });
                            };
                            var alternate_3 = ($$anchor8) => {
                              var fragment_12 = comment();
                              var node_7 = first_child(fragment_12);
                              {
                                var consequent_7 = ($$anchor9) => {
                                  TreeReport($$anchor9, {
                                    get report() {
                                      return get(report);
                                    },
                                    get locale() {
                                      return locale();
                                    },
                                    get operatingCurrencies() {
                                      return operatingCurrencies();
                                    },
                                    get renderCommas() {
                                      return renderCommas();
                                    }
                                  });
                                };
                                var alternate_2 = ($$anchor9) => {
                                  var fragment_14 = comment();
                                  var node_8 = first_child(fragment_14);
                                  {
                                    var consequent_8 = ($$anchor10) => {
                                      JournalReport($$anchor10, {
                                        get report() {
                                          return get(journal);
                                        },
                                        get renderCommas() {
                                          return renderCommas();
                                        }
                                      });
                                    };
                                    var alternate_1 = ($$anchor10) => {
                                      var fragment_16 = comment();
                                      var node_9 = first_child(fragment_16);
                                      {
                                        var consequent_9 = ($$anchor11) => {
                                          var title = derived_safe_equal(() => pageLabel(route()));
                                          GenericReport($$anchor11, {
                                            get report() {
                                              return get(table);
                                            },
                                            get title() {
                                              return get(title);
                                            },
                                            get route() {
                                              return route();
                                            },
                                            get locale() {
                                              return locale();
                                            },
                                            get renderCommas() {
                                              return renderCommas();
                                            }
                                          });
                                        };
                                        var alternate = ($$anchor11) => {
                                          var section_2 = root_20();
                                          var h2 = sibling(child(section_2), 2);
                                          var text_1 = child(h2, true);
                                          reset(h2);
                                          next(2);
                                          reset(section_2);
                                          template_effect(() => set_text(text_1, route()));
                                          append($$anchor11, section_2);
                                        };
                                        if_block(
                                          node_9,
                                          ($$render) => {
                                            if (get(table)) $$render(consequent_9);
                                            else $$render(alternate, false);
                                          },
                                          true
                                        );
                                      }
                                      append($$anchor10, fragment_16);
                                    };
                                    if_block(
                                      node_8,
                                      ($$render) => {
                                        if (get(journal)) $$render(consequent_8);
                                        else $$render(alternate_1, false);
                                      },
                                      true
                                    );
                                  }
                                  append($$anchor9, fragment_14);
                                };
                                if_block(
                                  node_7,
                                  ($$render) => {
                                    if (get(report)) $$render(consequent_7);
                                    else $$render(alternate_2, false);
                                  },
                                  true
                                );
                              }
                              append($$anchor8, fragment_12);
                            };
                            if_block(
                              node_6,
                              ($$render) => {
                                if (["options", "help", "diagnostics", "source"].includes(route())) $$render(consequent_6);
                                else $$render(alternate_3, false);
                              },
                              true
                            );
                          }
                          append($$anchor7, fragment_10);
                        };
                        if_block(
                          node_5,
                          ($$render) => {
                            if (route() === "import") $$render(consequent_5);
                            else $$render(alternate_4, false);
                          },
                          true
                        );
                      }
                      append($$anchor6, fragment_8);
                    };
                    if_block(
                      node_4,
                      ($$render) => {
                        if (route() === "editor") $$render(consequent_4);
                        else $$render(alternate_5, false);
                      },
                      true
                    );
                  }
                  append($$anchor5, fragment_6);
                };
                if_block(
                  node_3,
                  ($$render) => {
                    if (route() === "account") $$render(consequent_3);
                    else $$render(alternate_6, false);
                  },
                  true
                );
              }
              append($$anchor4, fragment_4);
            };
            if_block(
              node_2,
              ($$render) => {
                if (route() === "query") $$render(consequent_2);
                else $$render(alternate_7, false);
              },
              true
            );
          }
          append($$anchor3, fragment_2);
        };
        if_block(
          node_1,
          ($$render) => {
            if (get(error)) $$render(consequent_1);
            else $$render(alternate_8, false);
          },
          true
        );
      }
      append($$anchor2, fragment_1);
    };
    if_block(node, ($$render) => {
      if (get(loading)) $$render(consequent);
      else $$render(alternate_9, false);
    });
  }
  append($$anchor, fragment);
  pop();
}

// src/fava/keyboard-shortcuts.ts
function showTooltip(target2, description) {
  const { hidden } = target2;
  if (hidden) {
    target2.hidden = false;
  }
  const tooltip = document.createElement("div");
  tooltip.className = "keyboard-tooltip";
  tooltip.textContent = description;
  document.body.appendChild(tooltip);
  const targetRect = target2.getBoundingClientRect();
  const left = targetRect.left + Math.min((target2.offsetWidth - tooltip.offsetWidth) / 2, 10);
  const top = targetRect.top + (target2.offsetHeight - tooltip.offsetHeight) / 2;
  tooltip.style.left = `${left.toString()}px`;
  tooltip.style.top = `${(top + window.scrollY).toString()}px`;
  return () => {
    tooltip.remove();
    if (hidden) {
      target2.hidden = true;
    }
  };
}
function showTooltips() {
  const removes = [];
  document.querySelectorAll("[data-key]").forEach((el) => {
    const key = el.getAttribute("data-key");
    if (el instanceof HTMLElement && key != null) {
      removes.push(showTooltip(el, key));
    }
  });
  return () => {
    removes.forEach((r) => {
      r();
    });
  };
}
function isEditableElement(element2) {
  return element2 instanceof HTMLElement && (element2 instanceof HTMLInputElement || element2 instanceof HTMLSelectElement || element2 instanceof HTMLTextAreaElement || element2.isContentEditable);
}
var keyboardShortcuts = /* @__PURE__ */ new Map();
var lastChar = "";
function keydown(event2) {
  if (isEditableElement(event2.target)) {
    return;
  }
  let eventKey = event2.key;
  if (event2.metaKey) {
    eventKey = `Meta+${eventKey}`;
  }
  if (event2.altKey) {
    eventKey = `Alt+${eventKey}`;
  }
  if (event2.ctrlKey) {
    eventKey = `Control+${eventKey}`;
  }
  const lastTwoKeys = `${lastChar} ${eventKey}`;
  const handler = keyboardShortcuts.get(lastTwoKeys) ?? keyboardShortcuts.get(eventKey);
  if (handler) {
    if (handler instanceof HTMLInputElement) {
      event2.preventDefault();
      handler.focus();
    } else if (handler instanceof HTMLElement) {
      event2.preventDefault();
      handler.click();
    } else {
      handler(event2);
    }
  }
  if (event2.key !== "Alt" && event2.key !== "Control" && event2.key !== "Shift") {
    lastChar = eventKey;
  }
}
var isMac = (
  // eslint-disable-next-line @typescript-eslint/no-deprecated
  navigator.platform.startsWith("Mac") || navigator.platform === "iPhone"
);
function getKeySpecKey(spec) {
  if (typeof spec === "string") {
    return spec;
  }
  return isMac ? spec.mac ?? spec.key : spec.key;
}
function getKeySpecDescription(spec) {
  if (typeof spec === "string") {
    return spec;
  }
  const key = isMac ? spec.mac ?? spec.key : spec.key;
  return spec.note != null ? `${key} - ${spec.note}` : key;
}
function bindKey(spec, handler) {
  const key = getKeySpecKey(spec);
  const sequence = key.split(" ");
  if (sequence.length > 2) {
    console.error("Only key sequences of length <=2 are supported: ", key);
  }
  if (keyboardShortcuts.has(key)) {
    console.warn("Duplicate keyboard shortcut: ", key, handler);
  }
  keyboardShortcuts.set(key, handler);
  return () => {
    keyboardShortcuts.delete(key);
  };
}
function keyboardShortcut(node, spec) {
  if (spec == null) {
    return void 0;
  }
  node.setAttribute("data-key", getKeySpecDescription(spec));
  const unbind = bindKey(spec, node);
  return {
    destroy: () => {
      unbind();
      node.removeAttribute("data-key");
    }
  };
}
function initGlobalKeyboardShortcuts() {
  document.addEventListener("keydown", keydown);
  bindKey("?", () => {
    const hide = showTooltips();
    const once2 = () => {
      hide();
      document.removeEventListener("mousedown", once2);
      document.removeEventListener("keydown", once2);
      document.removeEventListener("scroll", once2);
    };
    document.addEventListener("mousedown", once2);
    document.addEventListener("keydown", once2);
    document.addEventListener("scroll", once2);
  });
}

// src/fava/components/Sidebar.svelte
var root_120 = template(`<div class="overlay svelte-16iwe3x" aria-hidden="true"></div>`);
var root_312 = template(`<li class="navigation-heading svelte-16iwe3x" aria-hidden="true"> </li>`);
var on_click3 = (event2, onNavigate, item) => {
  event2.preventDefault();
  onNavigate()(routeHref(get(item)));
};
var root_57 = template(`<li class="svelte-16iwe3x"><a class="svelte-16iwe3x"> </a></li>`);
var on_click_12 = (event2, onNavigate) => {
  event2.preventDefault();
  onNavigate()(routeHref("errors"));
};
var root_64 = template(`<ul class="navigation svelte-16iwe3x"><li class="svelte-16iwe3x"><a class="svelte-16iwe3x"> </a></li></ul>`);
var root_29 = template(`<ul class="navigation svelte-16iwe3x"><!> <!></ul> <!>`, 1);
var root11 = template(`<!> <div class="aside-buttons svelte-16iwe3x"><button id="menu-toggle" type="button" aria-controls="sidebar" aria-label="Menu" class="svelte-16iwe3x">\u2630</button> <a class="button svelte-16iwe3x" href="#add-transaction" aria-label="Add transaction">+</a></div> <aside id="sidebar" aria-label="Primary navigation" class="svelte-16iwe3x"></aside>`, 1);
function Sidebar($$anchor, $$props) {
  push($$props, false);
  let route = prop($$props, "route", 8);
  let open = prop($$props, "open", 8, false);
  let errors = prop($$props, "errors", 24, () => []);
  let locale = prop($$props, "locale", 8, "en");
  let onMenu = prop($$props, "onMenu", 8);
  let onNavigate = prop($$props, "onNavigate", 8);
  const sections = [
    [
      "",
      [
        "income_statement",
        "balance_sheet",
        "trial_balance",
        "journal",
        "query"
      ]
    ],
    [
      "",
      [
        "holdings",
        "commodities",
        "documents",
        "events",
        "statistics"
      ]
    ],
    [
      "",
      ["editor", "import", "options", "help"]
    ],
    ["OrangeCount", ["account"]]
  ];
  const known = /* @__PURE__ */ new Set([...ROUTES, "account"]);
  const shortcuts = {
    income_statement: "g i",
    balance_sheet: "g b",
    trial_balance: "g t",
    journal: "g j",
    query: "g q",
    holdings: "g h",
    commodities: "g c",
    documents: "g d",
    events: "g E",
    statistics: "g s",
    editor: "g e",
    import: "g n",
    options: "g o",
    help: "g H"
  };
  const keys = {
    income_statement: "incomeStatement",
    balance_sheet: "balanceSheet",
    trial_balance: "trialBalance",
    journal: "journal",
    query: "query",
    holdings: "holdings",
    commodities: "commodities",
    documents: "documents",
    events: "events",
    statistics: "statistics",
    editor: "editor",
    import: "import",
    options: "options",
    help: "help",
    account: "accounts"
  };
  function label(routeName) {
    const catalog = translations[locale() === "zh-CN" ? "zh-CN" : "en"];
    return catalog[keys[routeName] || ""] || pageLabel(routeName);
  }
  init();
  var fragment = root11();
  var node = first_child(fragment);
  {
    var consequent = ($$anchor2) => {
      var div = root_120();
      div.__click = function(...$$args) {
        onMenu()?.apply(this, $$args);
      };
      append($$anchor2, div);
    };
    if_block(node, ($$render) => {
      if (open()) $$render(consequent);
    });
  }
  var div_1 = sibling(node, 2);
  var button = child(div_1);
  button.__click = function(...$$args) {
    onMenu()?.apply(this, $$args);
  };
  next(2);
  reset(div_1);
  var aside = sibling(div_1, 2);
  each(aside, 5, () => sections, index, ($$anchor2, $$item, sectionIndex) => {
    let heading = () => get($$item)[0];
    let items = () => get($$item)[1];
    var fragment_1 = root_29();
    var ul = first_child(fragment_1);
    var node_1 = child(ul);
    {
      var consequent_1 = ($$anchor3) => {
        var li = root_312();
        var text2 = child(li, true);
        reset(li);
        template_effect(() => set_text(text2, heading()));
        append($$anchor3, li);
      };
      if_block(node_1, ($$render) => {
        if (heading()) $$render(consequent_1);
      });
    }
    var node_2 = sibling(node_1, 2);
    each(node_2, 1, items, index, ($$anchor3, item) => {
      var fragment_2 = comment();
      var node_3 = first_child(fragment_2);
      {
        var consequent_2 = ($$anchor4) => {
          var li_1 = root_57();
          var a = child(li_1);
          template_effect(() => set_attribute(a, "href", routeHref(get(item))));
          a.__click = [on_click3, onNavigate, item];
          var text_1 = child(a, true);
          template_effect(() => set_text(text_1, label(get(item))));
          reset(a);
          action(a, ($$node, $$action_arg) => keyboardShortcut?.($$node, $$action_arg), () => shortcuts[get(item)]);
          reset(li_1);
          template_effect(() => {
            set_attribute(a, "aria-current", route() === get(item) ? "page" : void 0);
            set_attribute(a, "data-route", get(item));
            toggle_class(a, "selected", route() === get(item));
          });
          append($$anchor4, li_1);
        };
        if_block(node_3, ($$render) => {
          if (known.has(get(item))) $$render(consequent_2);
        });
      }
      append($$anchor3, fragment_2);
    });
    reset(ul);
    var node_4 = sibling(ul, 2);
    {
      var consequent_3 = ($$anchor3) => {
        var ul_1 = root_64();
        var li_2 = child(ul_1);
        var a_1 = child(li_2);
        template_effect(() => set_attribute(a_1, "href", routeHref("errors")));
        a_1.__click = [on_click_12, onNavigate];
        var text_2 = child(a_1);
        reset(a_1);
        reset(li_2);
        reset(ul_1);
        template_effect(() => {
          set_attribute(a_1, "aria-current", route() === "errors" ? "page" : void 0);
          toggle_class(a_1, "selected", route() === "errors");
          set_text(text_2, `Errors (${errors().length ?? ""})`);
        });
        append($$anchor3, ul_1);
      };
      if_block(node_4, ($$render) => {
        if (sectionIndex === sections.length - 1 && errors().length) $$render(consequent_3);
      });
    }
    template_effect(() => set_attribute(ul, "aria-label", heading() || "Reports"));
    append($$anchor2, fragment_1);
  });
  reset(aside);
  template_effect(() => {
    toggle_class(div_1, "active", open());
    set_attribute(button, "aria-expanded", open());
    toggle_class(aside, "active", open());
  });
  append($$anchor, fragment);
  pop();
}
delegate(["click"]);

// node_modules/svelte/src/store/shared/index.js
var subscriber_queue = [];
function writable(value, start = noop) {
  let stop = null;
  const subscribers = /* @__PURE__ */ new Set();
  function set2(new_value) {
    if (safe_not_equal(value, new_value)) {
      value = new_value;
      if (stop) {
        const run_queue = !subscriber_queue.length;
        for (const subscriber of subscribers) {
          subscriber[1]();
          subscriber_queue.push(subscriber, value);
        }
        if (run_queue) {
          for (let i = 0; i < subscriber_queue.length; i += 2) {
            subscriber_queue[i][0](subscriber_queue[i + 1]);
          }
          subscriber_queue.length = 0;
        }
      }
    }
  }
  function update2(fn) {
    set2(fn(
      /** @type {T} */
      value
    ));
  }
  function subscribe(run2, invalidate = noop) {
    const subscriber = [run2, invalidate];
    subscribers.add(subscriber);
    if (subscribers.size === 1) {
      stop = start(set2, update2) || noop;
    }
    run2(
      /** @type {T} */
      value
    );
    return () => {
      subscribers.delete(subscriber);
      if (subscribers.size === 0 && stop) {
        stop();
        stop = null;
      }
    };
  }
  return { set: set2, update: update2, subscribe };
}

// src/fava/state.mjs
var DEFAULT_LOCALE = "en";
var DEFAULT_THEME = "system";
function stored(key, fallback2, values) {
  try {
    const value = localStorage.getItem(key);
    return values.includes(value) ? value : fallback2;
  } catch {
    return fallback2;
  }
}
function initialShellState(route) {
  return {
    route,
    locale: stored("orangecount-locale", DEFAULT_LOCALE, ["en", "zh-CN"]),
    theme: stored("orangecount-theme", DEFAULT_THEME, ["system", "dark", "light"]),
    loading: false,
    error: null,
    sidebarOpen: false,
    ledgerTitle: "OrangeCount",
    operatingCurrencies: [],
    renderCommas: false,
    query: {},
    revision: 0,
    errors: []
  };
}
function reduceShellState(state2, action2) {
  switch (action2.type) {
    case "route":
      return { ...state2, route: action2.route, query: action2.query || {}, sidebarOpen: false, error: null };
    case "locale":
      return { ...state2, locale: action2.locale === "zh-CN" ? "zh-CN" : DEFAULT_LOCALE };
    case "theme":
      return { ...state2, theme: ["system", "dark", "light"].includes(action2.theme) ? action2.theme : DEFAULT_THEME };
    case "account":
      return { ...state2, account: action2.account || "" };
    case "query":
      return { ...state2, query: { ...state2.query, ...action2.query } };
    case "menu":
      return { ...state2, sidebarOpen: action2.open ?? !state2.sidebarOpen };
    case "loading":
      return { ...state2, loading: Boolean(action2.value), error: action2.value ? null : state2.error };
    case "error":
      return { ...state2, loading: false, error: action2.message || "The local adapter could not load this view." };
    case "clear-error":
      return { ...state2, error: null };
    case "bootstrap":
      return {
        ...state2,
        ledgerTitle: action2.ledgerTitle || state2.ledgerTitle,
        operatingCurrencies: Array.isArray(action2.operatingCurrencies) ? action2.operatingCurrencies : state2.operatingCurrencies,
        renderCommas: typeof action2.renderCommas === "boolean" ? action2.renderCommas : state2.renderCommas,
        locale: action2.locale === "zh-CN" ? "zh-CN" : state2.locale,
        theme: ["system", "dark", "light"].includes(action2.theme) ? action2.theme : state2.theme,
        errors: Array.isArray(action2.errors) ? action2.errors : state2.errors,
        error: null,
        loading: false,
        revision: state2.revision + 1
      };
    default:
      return state2;
  }
}
function createShellStore(initial) {
  const store = writable(initial);
  return {
    subscribe: store.subscribe,
    dispatch(action2) {
      store.update((state2) => reduceShellState(state2, action2));
    }
  };
}

// src/fava/App.svelte
var root_121 = template(`<meta name="description" content="OrangeCount local ledger interface">`);
var root12 = template(`<!> <!> <article id="main-content" tabindex="-1"><!></article>`, 1);
function App($$anchor, $$props) {
  push($$props, false);
  const $$stores = setup_stores();
  const $shell = () => store_get(shell, "$shell", $$stores);
  const current = mutable_state();
  const initialRoute = parseRoute(window.location.href);
  const shell = createShellStore({
    ...initialShellState(initialRoute.route),
    account: initialRoute.account,
    query: initialRoute.query
  });
  const adapter = createAdapterClient();
  function navigate(href) {
    const target2 = new URL(href, window.location.href);
    const next2 = parseRoute(target2.href);
    window.history.pushState({}, "", target2.href);
    shell.dispatch({
      type: "route",
      route: next2.route,
      query: next2.query
    });
    shell.dispatch({ type: "account", account: next2.account });
  }
  function setQuery(value) {
    const href = updateQuery(window.location.href, { filter: value });
    const target2 = new URL(href, window.location.href);
    window.history.replaceState({}, "", target2.href);
    shell.dispatch({ type: "query", query: { filter: value } });
  }
  function setAccount(value) {
    const href = updateQuery(window.location.href, { account: value });
    const target2 = new URL(href, window.location.href);
    window.history.replaceState({}, "", target2.href);
    shell.dispatch({ type: "query", query: { account: value } });
  }
  function setTime(value) {
    const href = updateQuery(window.location.href, { time: value });
    const target2 = new URL(href, window.location.href);
    window.history.replaceState({}, "", target2.href);
    shell.dispatch({ type: "query", query: { time: value } });
  }
  function setConversion(value) {
    const href = updateQuery(window.location.href, { conversion: value });
    const target2 = new URL(href, window.location.href);
    window.history.replaceState({}, "", target2.href);
    shell.dispatch({ type: "query", query: { conversion: value } });
  }
  function setInterval(value) {
    const href = updateQuery(window.location.href, { interval: value });
    const target2 = new URL(href, window.location.href);
    window.history.replaceState({}, "", target2.href);
    shell.dispatch({ type: "query", query: { interval: value } });
  }
  function setLocale(locale) {
    try {
      localStorage.setItem("orangecount-locale", locale);
    } catch {
    }
    shell.dispatch({ type: "locale", locale });
  }
  function setTheme(theme) {
    try {
      localStorage.setItem("orangecount-theme", theme);
    } catch {
    }
    shell.dispatch({ type: "theme", theme });
  }
  async function bootstrap() {
    shell.dispatch({ type: "loading", value: true });
    try {
      const payload = await adapter.bootstrap();
      shell.dispatch({
        type: "bootstrap",
        ledgerTitle: payload.ledger_title,
        locale: payload.locale,
        theme: payload.theme,
        errors: payload.errors,
        operatingCurrencies: payload.operating_currencies,
        renderCommas: payload.render_commas
      });
    } catch (error) {
      notify_err(error);
      shell.dispatch({
        type: "error",
        message: error instanceof Error ? error.message : "The local adapter could not load this view."
      });
    }
  }
  onMount(() => {
    initGlobalKeyboardShortcuts();
    const onPopState = () => {
      const next2 = parseRoute(window.location.href);
      shell.dispatch({
        type: "route",
        route: next2.route,
        query: next2.query
      });
      shell.dispatch({ type: "account", account: next2.account });
    };
    window.addEventListener("popstate", onPopState);
    void bootstrap();
    const poll = window.setInterval(
      async () => {
        try {
          if (await adapter.changed()) await bootstrap();
        } catch {
        }
      },
      5e3
    );
    return () => {
      window.removeEventListener("popstate", onPopState);
      window.clearInterval(poll);
    };
  });
  function retry() {
    void bootstrap();
  }
  legacy_pre_effect(() => $shell(), () => {
    set(current, $shell());
  });
  legacy_pre_effect(() => get(current), () => {
    document.documentElement.lang = get(current).locale;
  });
  legacy_pre_effect(() => get(current), () => {
    document.documentElement.dataset.theme = get(current).theme === "system" ? "" : get(current).theme;
  });
  legacy_pre_effect(() => get(current), () => {
    document.documentElement.style.colorScheme = get(current).theme === "system" ? "light dark" : get(current).theme;
  });
  legacy_pre_effect_reset();
  init();
  var fragment = root12();
  head(($$anchor2) => {
    var meta = root_121();
    template_effect(() => $document.title = `${get(current).ledgerTitle ?? ""} \u203A ${(get(current).account || get(current).route) ?? ""}`);
    append($$anchor2, meta);
  });
  var node = first_child(fragment);
  var time = derived_safe_equal(() => get(current).query.time || "");
  var accountFilter = derived_safe_equal(() => get(current).query.account || "");
  var filter = derived_safe_equal(() => get(current).query.filter || "");
  var conversion = derived_safe_equal(() => get(current).query.conversion || "at_cost");
  var interval = derived_safe_equal(() => get(current).query.interval || "month");
  Header(node, {
    get ledgerTitle() {
      return get(current).ledgerTitle;
    },
    get route() {
      return get(current).route;
    },
    get account() {
      return get(current).account;
    },
    get locale() {
      return get(current).locale;
    },
    get time() {
      return get(time);
    },
    get accountFilter() {
      return get(accountFilter);
    },
    get filter() {
      return get(filter);
    },
    onNavigate: navigate,
    onTime: setTime,
    onAccount: setAccount,
    get conversion() {
      return get(conversion);
    },
    get interval() {
      return get(interval);
    },
    onConversion: setConversion,
    onInterval: setInterval,
    onQuery: setQuery
  });
  var node_1 = sibling(node, 2);
  Sidebar(node_1, {
    get route() {
      return get(current).route;
    },
    get open() {
      return get(current).sidebarOpen;
    },
    get errors() {
      return get(current).errors;
    },
    get locale() {
      return get(current).locale;
    },
    onMenu: () => shell.dispatch({ type: "menu" }),
    onNavigate: navigate
  });
  var article = sibling(node_1, 2);
  var node_2 = child(article);
  LoadingBoundary(node_2, {
    get active() {
      return get(current).loading;
    },
    children: ($$anchor2, $$slotProps) => {
      ErrorBoundary($$anchor2, {
        get message() {
          return get(current).error;
        },
        onRetry: retry,
        children: ($$anchor3, $$slotProps2) => {
          var fragment_2 = comment();
          var node_3 = first_child(fragment_2);
          key_block(node_3, () => get(current).revision, ($$anchor4) => {
            var query = derived_safe_equal(() => ({
              ...get(current).query,
              ...get(current).account ? { account: get(current).account } : {}
            }));
            ReportOutlet($$anchor4, {
              adapter,
              get route() {
                return get(current).route;
              },
              get locale() {
                return get(current).locale;
              },
              get theme() {
                return get(current).theme;
              },
              get operatingCurrencies() {
                return get(current).operatingCurrencies;
              },
              get renderCommas() {
                return get(current).renderCommas;
              },
              onLocale: setLocale,
              onTheme: setTheme,
              get query() {
                return get(query);
              }
            });
          });
          append($$anchor3, fragment_2);
        },
        $$slots: { default: true }
      });
    },
    $$slots: { default: true }
  });
  reset(article);
  append($$anchor, fragment);
  pop();
}

// src/fava/main.ts
var target = document.getElementById("app");
if (!target) throw new Error("Fava shell mount target is missing");
mount(App, { target });
