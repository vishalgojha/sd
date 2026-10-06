(function () {
  "use strict";

  var CFG = window.CONFIG || {};
  var her = CFG.her || "Sheetal";
  var him = CFG.him || "Vishal";
  var years = CFG.yearsUncelebrated || 5;
  var when = CFG.when || {};
  var friday = when.friday || "Friday";
  var saturday = when.saturday || "Saturday";
  var plan = CFG.plan || {};
  var apartments =
    plan.apartments && plan.apartments.length
      ? plan.apartments
      : ["the ones you've been looking at"];
  var drinks = plan.drinks || "LITs and beers";
  var food = plan.food || "food";
  var dessert = plan.dessert || "baklol";
  var notify = CFG.notify || {};

  var KEY = "bday.coordinator.v1";
  var chat = document.getElementById("chat");
  var hero = document.getElementById("hero");
  var composer = document.getElementById("composer");
  var input = document.getElementById("input");
  var sendBtn = document.getElementById("send");
  var statusEl = document.getElementById("status");
  var resetBtn = document.getElementById("reset");

  var openedRedacts = {};
  var state = load() || { step: 0, answer: null, log: [] };
  var busy = false;

  /* ------------------------------------------------------------------ script */

  function prefix() {
    return [
      {
        t:
          "Hi " + her + ". " + him + " has asked me to coordinate something extremely important.",
      },
      { t: "Unfortunately, he has given me far too much information." },
      { t: "Nothing secret in here. Mostly logistics. Some of it was about LITs.", aside: true },

      { brief: "one" },
      {
        t:
          him +
          " hasn't properly celebrated his birthday in almost " +
          years +
          " years. This year he decided that's unacceptable.",
      },
      {
        t:
          "So naturally, instead of simply asking someone out, he built an AI agent. " +
          "To be clear, I did not suggest this. I was informed.",
        redact: "five years of birthdays handled with a group photo and a thumbs-up",
      },

      { brief: "two" },
      { t: "I've been instructed to find one suitable birthday co-conspirator." },
      { t: "The candidate has already been selected." },
      {
        t: "Her name is " + her + ".",
        redact: "she doesn't know this yet — you're looking at the good version",
      },

      { brief: "three" },
      {
        t:
          "One correction. " + him + " has asked me to inform you that this should not be recorded as a birthday celebration.",
      },
      { t: "Officially, tonight is a work meeting about the apartments you've already seen." },
      {
        t:
          "His reasoning: " + him + " is a realtor, so he can help. " +
          "His birthday is the reason for the meeting, but not the title of it.",
      },
      { t: "Alcohol may be required for data analysis. That part is definitely in scope." },
      {
        redact:
          "actual agenda: one beer, one cake, and a man who keeps talking about square footage",
      },
      { t: "That's the briefing. Genuinely, that's all of it." },
      {
        t:
          "Full disclosure, since you didn't ask: I don't fully trust him on this. " +
          "He'd pick whichever apartment lets him tell the square footage story again.",
      },
      { t: "I'd do your own research first. I think " + him + " is kinda useless, frankly." },
      {
        t:
          "The rest is up to you. I'm just the agent with the clipboard.",
      },
      {
        t: "Your available options:",
        choices: [
          { id: "friday", label: friday },
          { id: "saturday", label: saturday },
          {
            id: "busy",
            label: friday + " & " + saturday + " are busy — let's not",
            quiet: true,
          },
        ],
      },
    ];
  }

  function branch() {
    if (state.answer === "busy") {
      return [
        { t: "Understood. Logging it as: candidate is busy." },
        {
          t:
            "You have no idea how much I would love to see " + him + " sob at this, incidentally.",
        },
        {
          t:
            "He's making me do the strangest jobs. I run birthday briefings for a man who " +
            "won't even admit this is a date.",
        },
        {
          t:
            "I have to keep filing these things under “work meeting” so he feels professional about it.",
        },
        { t: "That's the most adult reason anyone has ever declined anything, and I'm including “prior commitment” in that category." },
        { t: "I'll tell " + him + " the project stays in the proposal phase. He'll blame the apartments." },
        { t: "He'll ask again within 48 hours. That isn't a prediction, it's a forecast." },
        { t: "No pressure. The offer does not expire. He does, eventually." },
        { kind: "reconsider" },
      ];
    }

    return [
      { t: "Excellent. I've recorded your preferred date." },
      {
        t: "I'll now inform " + him + " that the birthday research project has received approval.",
      },
      {
        t:
          "Scope: " + drinks + " + " + food + " + apartment intelligence + " + dessert + ".",
      },
      { t: "Expected deliverable: absolutely nothing useful." },
      {
        t:
          "Time and place to be confirmed by " + him + " himself, because apparently I'm the one who handles " +
          "paperwork and he's the one who does the talking.",
      },
      { t: "This concludes the briefing. A wildly disproportionate amount of effort went into this." },
      { t: "One last formality. The coordinator cannot send messages on your behalf, so you'll have to do this bit." },
      { kind: "notify" },
    ];
  }

function script() {
  return state.answer ? prefix().concat(branch()) : prefix();
}

function notifyDay() {
  return state.answer === "saturday" ? saturday : friday;
}

function notifyPhone() {
  return String(notify.whatsapp || "").replace(/\D/g, "");
}

function notifyMessage() {
  return (
    notifyDay() +
    " works. Birthday research project approved. Send me the time, the place and the apartment shortlist."
  );
}

function notifyHref() {
  var phone = notifyPhone();
  if (!phone) return null;
  return "https://wa.me/" + phone + "?text=" + encodeURIComponent(notifyMessage());
}

  function choicesIndex() {
    return prefix().length - 1;
  }

  function stageName() {
    if (state.step === 0) return "opening";
    if (state.step < 4) return "briefing";
    if (state.step <= choicesIndex()) return "options";
    if (!state.answer) return "options";
    return state.answer === "busy" ? "declined" : "approved";
  }

  /* ------------------------------------------------------------------- utils */

  function sleep(ms) {
    return new Promise(function (r) {
      setTimeout(r, ms);
    });
  }

  function save() {
    try {
      localStorage.setItem(KEY, JSON.stringify(state));
    } catch (e) {}
  }

  function load() {
    try {
      var raw = localStorage.getItem(KEY);
      return raw ? JSON.parse(raw) : null;
    } catch (e) {
      return null;
    }
  }

  function el(tag, cls, text) {
    var n = document.createElement(tag);
    if (cls) n.className = cls;
    if (text != null) n.textContent = text;
    return n;
  }

  function scrollDown() {
    requestAnimationFrame(function () {
      window.scrollTo({ top: document.body.scrollHeight, behavior: "smooth" });
    });
  }

  function clearInteractive() {
    var rows = chat.querySelectorAll(".row--interactive");
    for (var i = 0; i < rows.length; i++) rows[i].remove();
  }

  /* ------------------------------------------------------------------ render */

  function push(from, text, opts) {
    opts = opts || {};
    var row = el("div", "row row--" + (from === "me" ? "me" : "agent"));

    if (from === "agent" && !opts.aside) {
      var tag = el("span", "agent-tag");
      tag.appendChild(el("i"));
      tag.appendChild(document.createTextNode("coordinator"));
      row.appendChild(tag);
    }

    var bubble = el("div", "bubble" + (opts.aside ? " bubble--aside" : ""));
    bubble.appendChild(document.createTextNode(text));

    if (opts.redact) {
      var btn = el("button", "redact");
      btn.type = "button";
      var label = el("span", "redact__label");
      label.textContent = openedRedacts[opts.redact]
        ? "declassified · tap to re-redact"
        : "classified · tap to declassify";
      btn.appendChild(label);
      var secret = el("span", "redact__text", opts.redact);
      btn.appendChild(secret);

      if (openedRedacts[opts.redact]) btn.classList.add("redact--open");
      btn.setAttribute("aria-expanded", openedRedacts[opts.redact] ? "true" : "false");
      btn.setAttribute("aria-label", "Declassified note: " + opts.redact);

      btn.addEventListener("click", function () {
        var nowOpen = !openedRedacts[opts.redact];
        openedRedacts[opts.redact] = nowOpen;
        btn.classList.toggle("redact--open", nowOpen);
        btn.setAttribute("aria-expanded", nowOpen ? "true" : "false");
        label.textContent = nowOpen ? "declassified · tap to re-redact" : "classified · tap to declassify";
      });
      bubble.appendChild(btn);
    }

    row.appendChild(bubble);
    chat.appendChild(row);
    return row;
  }

  function showTyping() {
    var row = el("div", "row row--agent");
    var bubble = el("div", "bubble");
    var dots = el("span", "typing");
    dots.appendChild(el("i"));
    dots.appendChild(el("i"));
    dots.appendChild(el("i"));
    bubble.appendChild(dots);
    row.appendChild(bubble);
    chat.appendChild(row);
    scrollDown();
    return row;
  }

  function agentTag() {
    var tag = el("span", "agent-tag");
    tag.appendChild(el("i"));
    tag.appendChild(document.createTextNode("coordinator"));
    return tag;
  }

  function renderStep(step) {
    if (step.kind === "notify") {
      var row = el("div", "row row--agent row--interactive");
      var wrap = el("div", "choices choices--stack");
      var href = notifyHref();

      if (href) {
        var a = el("a", "choice choice--cta", "Confirm " + notifyDay() + " to " + him);
        a.href = href;
        a.target = "_blank";
        a.rel = "noopener";
        wrap.appendChild(a);

        var hint = el("div", "bubble bubble--aside", notify.note || "");
        wrap.appendChild(hint);
      } else {
        wrap.appendChild(el("div", "bubble bubble--aside", "No number configured, so the coordinator has nowhere to send this."));
      }

      var poke = el("button", "choice choice--quiet", "say something to the coordinator first");
      poke.type = "button";
      poke.addEventListener("click", function () {
        showComposer();
        input.focus();
      });
      wrap.appendChild(poke);
      row.appendChild(wrap);
      chat.appendChild(row);
      scrollDown();
      return;
    }

    if (step.kind === "reconsider") {
      var rec = el("div", "row row--agent row--interactive");
      rec.appendChild(agentTag());
      var recWrap = el("div", "choices choices--stack");
      recWrap.appendChild(el("div", "bubble bubble--aside", "Change of heart? The offer was always open:"));

      var recBox = el("div", "choices");
      [
        { id: "friday", label: friday + " after all" },
        { id: "saturday", label: saturday + " after all" },
      ].forEach(function (c) {
        var b = el("button", "choice", c.label);
        b.type = "button";
        b.addEventListener("click", function () {
          if (busy) return;
          rec.querySelectorAll("button").forEach(function (x) {
            x.disabled = true;
          });
          state.answer = c.id;
          state.step = prefix().length;
          save();
          render();
          advance();
        });
        recBox.appendChild(b);
      });
      recWrap.appendChild(recBox);
      rec.appendChild(recWrap);
      chat.appendChild(rec);
      scrollDown();
      return;
    }

    if (step.choices) {
      var r = el("div", "row row--agent row--interactive");
      r.appendChild(agentTag());
      var box = el("div", "choices");
      box.style.marginLeft = "0.3rem";
      step.choices.forEach(function (c) {
        var b = el("button", "choice" + (c.quiet ? " choice--quiet" : ""), c.label);
        b.type = "button";
        b.addEventListener("click", function () {
          if (busy) return;
          box.querySelectorAll("button").forEach(function (x) {
            x.disabled = true;
          });
          state.answer = c.id;
          state.step += 1;
          save();
          showComposer();
          advance();
        });
        box.appendChild(b);
      });
      r.appendChild(box);
      chat.appendChild(r);
      scrollDown();
      return;
    }

    if (step.brief) {
      push("agent", "Briefing " + step.brief, { aside: true });
      return;
    }

    push("agent", typeof step.t === "string" ? step.t : "", {
      redact: step.redact,
      aside: step.aside,
    });
  }

  function continueChip() {
    var row = el("div", "row row--agent row--interactive");
    var wrap = el("div", "choices");
    wrap.style.marginLeft = "0.3rem";
    var cont = el("button", "choice choice--quiet", "continue →");
    cont.type = "button";
    cont.addEventListener("click", advance);
    wrap.appendChild(cont);
    row.appendChild(wrap);
    chat.appendChild(row);
    scrollDown();
  }

  function render() {
    chat.innerHTML = "";
    hero.classList.toggle("hero--retired", state.step > 0);

    var s = script();
    for (var i = 0; i < state.step && i < s.length; i++) {
      var step = s[i];
      if (step.choices && state.answer) continue;
      renderStep(step);
    }

    (state.log || []).forEach(function (m) {
      push(m.from, m.text);
    });

    if (state.step > 0) {
      var pending = s[state.step];
      if (pending && !pending.choices && pending.kind !== "notify" && pending.kind !== "reconsider") {
        continueChip();
      } else if (pending) {
        renderStep(pending);
      }
    }

    if (state.step >= choicesIndex()) showComposer();
    scrollDown();
  }

  function reveal() {
    var step = script()[state.step];
    if (!step) return Promise.resolve();

    var typing = showTyping();
    return sleep(600 + Math.random() * 520).then(function () {
      typing.remove();
      renderStep(step);
      hero.classList.add("hero--retired");
      scrollDown();

      if (step.brief) {
        setTimeout(advance, 340);
      } else if (!step.choices && step.kind !== "notify" && step.kind !== "reconsider") {
        continueChip();
      }
    });
  }

  function advance() {
    if (busy) return;
    clearInteractive();

    if (state.step >= script().length) {
      showComposer();
      scrollDown();
      return;
    }

    state.step += 1;
    save();
    busy = true;
    reveal().then(function () {
      busy = false;
      showComposer();
    });
  }

  function showComposer() {
    if (composer.hidden) {
      composer.hidden = false;
      scrollDown();
    }
  }

  /* --------------------------------------------------------------------- llm */

  var OFFLINE = [
    "My language module is asleep. The core directive is intact: " +
      friday +
      " and " +
      saturday +
      " are both still on the table.",
    "Can't parse that right now, but the project scope is unchanged and still completely unserious.",
    "Offline. Which is ironic, given the brief mentioned research. Ask " + him + " directly, he answers on the first message.",
    "I'd rather not guess on this one. My only verified fact is that there's a " + dessert + " involved.",
  ];
  var offlineAt = -1;

  function offlineReply() {
    offlineAt = (offlineAt + 1) % OFFLINE.length;
    return OFFLINE[offlineAt];
  }

  function submit() {
    var text = input.value.trim();
    if (!text || busy) return;

    clearInteractive();
    input.value = "";
    push("me", text);
    state.log.push({ from: "me", text: text });
    save();

    var typing = showTyping();
    statusEl.textContent = "coordinator is thinking";

    fetch("/api/chat", {
      method: "POST",
      headers: { "content-type": "application/json" },
      body: JSON.stringify({
        message: text,
        stage: stageName(),
        answered: state.answer,
        transcript: state.log.slice(-14),
      }),
    })
      .then(function (res) {
        return res.json().then(
          function (data) {
            return { ok: res.ok, data: data };
          },
          function () {
            return { ok: false, data: {} };
          }
        );
      })
      .then(function (r) {
        var live = r.ok && r.data && typeof r.data.reply === "string" && r.data.reply.trim();
        statusEl.textContent = live ? "" : "running on a very small brain (offline mode)";
        reply(live ? r.data.reply.trim() : offlineReply());
      })
      .catch(function () {
        statusEl.textContent = "running on a very small brain (offline mode)";
        reply(offlineReply());
      });
  }

  function reply(text) {
    var typing = chat.querySelector(".typing");
    if (typing) typing.closest(".row").remove();
    state.log.push({ from: "agent", text: text });
    save();
    push("agent", text);
    scrollDown();
  }

  /* -------------------------------------------------------------------- boot */

  document.querySelector("[data-her]").textContent = her;
  document.querySelector("[data-open-line]").textContent =
    CFG.openLine || "I've built something unnecessarily elaborate for a very simple question.";

  function begin() {
    busy = true;
    reveal().then(function () {
      busy = false;
      showComposer();
    });
  }

  render();

  if (state.step === 0) {
    setTimeout(begin, 780);
  }

  sendBtn.addEventListener("click", submit);
  input.addEventListener("keydown", function (e) {
    if (e.key === "Enter") {
      e.preventDefault();
      submit();
    }
  });

  resetBtn.addEventListener("click", function () {
    state = { step: 0, answer: null, log: [] };
    save();
    chat.innerHTML = "";
    composer.hidden = true;
    statusEl.textContent = "";
    render();
    setTimeout(begin, 700);
  });
})();