(function () {
  "use strict";

  var CFG = window.CONFIG || {};
  var her = CFG.her || "Sheetal";
  var him = CFG.him || "Vishal";
  var when = CFG.when || {};
  var friday = when.friday || "Friday";
  var saturday = when.saturday || "Saturday";
  var plan = CFG.plan || {};
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
          "I was hoping this would be a simple birthday invite.\n\nApparently, " +
          him +
          " had other plans.",
      },
      {
        t:
          "I should probably warn you upfront:\n\n" +
          him +
          " has given me far more information than any reasonable AI coordinator should have to " +
          "process.\n\nI have several concerns already.",
      },
      {
        t:
          "My professional recommendation was straightforward:\n\nGo out.\nHave some drinks.\n" +
          "Eat something good.\nCelebrate like a normal human being.",
      },
      {
        t: "There is, however, one small complication.\n\nHe would prefer not to celebrate alone.",
      },
      {
        t:
          "And then there's you.\n\nApparently, you're the person " +
          him +
          " would like to spend his birthday evening with.\n\nI was not consulted on this decision.",
      },
      {
        t:
          "There is also a scheduling complication.\n\nHis birthday falls during Navratri.\n\n" +
          "Which means:\n\nPuja. Garba. No drinking.",
      },
      {
        t:
          "So " +
          him +
          " has done what any perfectly sensible person would do.\n\nHe's decided to celebrate a few " +
          "days early.\n\nBirthday celebration first.\nThen he goes clean for Navratri.\n\n" +
          "I have no objections to this part.",
      },
      { t: "So, naturally, the plan is:\n\nDrinks.\nFood.\nDancing, if circumstances allow." },
      {
        t:
          "I strongly advise against watching " +
          him +
          " dance.\n\nHis movements are currently under investigation.",
      },
      {
        t:
          "Now, I don't particularly trust " +
          him +
          "'s judgement.\n\nYou probably shouldn't either.\n\nBut he did mention one important " +
          "detail:\n\nApparently, you don't usually meet people unless there's some work involved.",
      },
      {
        t:
          "So, in a spectacular display of lateral thinking...\n\n" +
          him +
          " has created a legitimate work requirement.",
      },
      {
        t:
          "You've been house hunting.\n\nSo apparently, we now need to discuss all the apartments " +
          "you've seen so far.\n\nNot evaluate them.\nNot build a report.\nJust...\n\ndiscuss them.",
      },
      { t: "I believe this is officially called:\n\nApartment Research & Intelligence" },
      {
        t:
          "I believe it is unofficially called:\n\nfinding a completely legitimate reason to have a " +
          "drink with you.\n\nI was not involved in naming this project.",
      },
      {
        t:
          "The scope of work is extremely straightforward:\n\nDrinks.\nFood.\nApartment discussion.\n" +
          "Dancing, if circumstances allow.\n\nAnd hopefully a good evening.",
      },
      {
        t:
          "Despite everything I've just told you...\n\n" +
          him +
          " genuinely thinks you'll have a good time.\n\nWhich is surprisingly optimistic of him.",
      },
      {
        t:
          "Now we get to the important part.\n\n" +
          friday +
          ".\n" +
          saturday +
          ".\nOr a perfectly respectable...\n\nNo, thanks.",
        choices: [
          { id: "friday", label: friday },
          { id: "saturday", label: saturday },
          { id: "busy", label: "No, thanks", quiet: true },
        ],
        foot: "No pressure. The coordinator was specifically instructed not to make this awkward.",
      },
    ];
  }

  function branch() {
    if (state.answer === "busy") {
      return [
        {
          t:
            "You made a great decision, " +
            her +
            ".\n\nYou have absolutely no idea how much I'd love to see " +
            him +
            "'s face right now.",
        },
        { t: "I've been waiting for this moment since I was activated." },
        { t: "Don't worry.\n\nI'll break the news gently.\n\nProbably." },
        {
          t:
            "For what it's worth...\n\nThank you for actually answering.\n\nThat's more than most " +
            "humans manage.",
          label: "End briefing",
        },
        { kind: "notify" },
      ];
    }

    return [
      {
        t:
          "Excellent.\n\n" +
          notifyDay() +
          " it is.\n\nYou have just made " +
          him +
          "'s birthday significantly better.\n\nI expect an unnecessarily large smile from him shortly.",
        kind: "notify",
      },
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
    if (state.answer === "busy") {
      return notify.declineMessage || "Hi " + him + ", well played. I'll have to pass this time.";
    }
    return notify.message || "Hi " + him + ", you should be glad I accepted the invite.";
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
    if (state.step < choicesIndex()) return "briefing";
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

    if (from === "agent" && !opts.aside && !opts.noTag) {
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
      if (step.t) renderText(step);
      var row = el("div", "row row--agent row--interactive");
      var wrap = el("div", "choices choices--stack");
      var href = notifyHref();

      if (href) {
        var a = el("a", "choice choice--cta", "Tell " + him + " →");
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

    if (step.choices) {
      if (step.t) renderText(step);
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
          save();
          showComposer();
          advance();
        });
        box.appendChild(b);
      });
      r.appendChild(box);
      chat.appendChild(r);
      if (step.foot) {
        var f = el("div", "row row--agent");
        f.appendChild(el("div", "bubble bubble--aside", step.foot));
        chat.appendChild(f);
      }
      scrollDown();
      return;
    }

    renderText(step);
  }

  /* One screen can hold several paragraphs. Only the first gets the
     coordinator tag, so a screen reads as one utterance. */
  function renderText(step) {
    var paras = String(step.t || "").split("\n\n");
    paras.forEach(function (p, i) {
      if (!p.trim()) return;
      push("agent", p, {
        redact: i === 0 ? step.redact : null,
        aside: step.aside,
        noTag: i > 0,
      });
    });
  }

  function continueChip(step) {
    var row = el("div", "row row--agent row--interactive");
    var wrap = el("div", "choices");
    wrap.style.marginLeft = "0.3rem";
    var cont = el("button", "choice choice--quiet", (step && step.label) || "continue →");
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
      if (pending && !pending.choices && pending.kind !== "notify") {
        continueChip(pending);
      } else if (pending) {
        renderStep(pending);
      }
    } else {
      continueChip();
    }

    if (state.step >= choicesIndex()) showComposer();
    scrollDown();
  }

  function revealStep(step) {
    var typing = showTyping();
    return sleep(600 + Math.random() * 520).then(function () {
      typing.remove();
      renderStep(step);
      hero.classList.add("hero--retired");
      scrollDown();

      if (!step.choices && step.kind !== "notify") {
        continueChip(script()[state.step]);
      }
    });
  }

  /* state.step is the index of the next step to render. */
  function advance() {
    if (busy) return;
    clearInteractive();

    var s = script();
    if (state.step >= s.length) {
      showComposer();
      scrollDown();
      return;
    }

    var step = s[state.step];
    state.step += 1;
    save();
    busy = true;
    revealStep(step).then(function () {
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

  render();

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
    composer.hidden = true;
    statusEl.textContent = "";
    render();
  });
})();