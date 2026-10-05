# The Method

Adapted from Katie Dill (Stripe), "How to scale intent, quality, and artistry with AI", Lenny & Friends Summit, 2026 — https://www.youtube.com/watch?v=GLvFTMtw4Jk

This is a summary in our own words, with short quotes from the talk. Read it when a question needs its reasoning, or when the user asks what the method says.

## Contents

1. The problem: zombie UI
2. Three warning signs
3. Four principles
4. Named tests
5. Tactics and culture

## 1. The problem: zombie UI

After the Second World War, builders copied modernism's look without the thinking behind it, and produced decades of interchangeable "zombie buildings": generic, and a poor fit for where they stood. AI has started a new building boom, and it risks producing the same thing in software: zombie UI that is monotonous, empty, or uncared for. The talk's examples: nine interchangeable landing pages ("Ship faster. Sleep better."), and a Korean barbecue restaurant whose site looks like a SaaS product.

## 2. Three warning signs

- **The most probable answer.** LLMs are good at what was popular and has worked before. They are poor at what is specific to you, your brand, and your users' context.
- **The temptation of done.** AI makes work look finished quickly, often too early. It's the microwave burrito: lunch in 90 seconds, and the speed makes you forgive that it's barely edible. Ask whether it actually solves the problem and stands apart.
- **Disposability.** Work that is quick to make gets treated as throwaway, and nobody asks who will maintain it.

## 3. Four principles

**1. Have a point of view.** "If you don't, AI will give it to you", and it will be generic and backward-looking. Decide what the work is for you, what it is for your users, who you want to be to them, and what they care about. That is the basis of your standards. When building is spread across many people and agents, ownership gets spread too, and it becomes easy to hand decisions to the AI. Users don't care how the work was made, only whether it's good, so the bar for AI-made work is the same bar. Aim one level deeper than the customer can see; the goal is that level, not perfection. Taste comes from noticing: what users need (not only what they say), and what separates great from "meh" in products, art and science.

**2. Encode your standards into the machine.** People won't be in the room for most decisions. Interfaces get built without a designer present, agents fix problems overnight, and some UI is generated at runtime. "The old system scaled consistency, but the new system needs to scale intent." Stripe's first attempt was an MCP server over its design docs, and the same prompt gave three people three different results. What worked was a CLI where builders already work. It serves structured docs for one item at a time, checks code against the system, and ships full templates, flows, decision trees ("single short action → dialog") and do/don't pairs, not just components. But "a system can satisfy every rule and still be dead" (Christopher Alexander). Rules only set the floor.

**3. Refuse to confuse done with good.** Quality used to be filtered before building: there were twenty ideas and staff for one. Now twenty can be built in a week, so the filter has to run after the build, when saying no is harder. Someone has to be the editor. Unlearn two assumptions: that built means done, and that done means good.
- Use the work as a user would. Does it solve the problem, and does it fit how they think?
- Check that it holds together, because parts that are fine alone can feel disconnected end to end.
- Ask whether it is fully formed, not just yes or no.

What AI does badly is what makes work great: unexpected details, deeper meaning, and themes that tie it into a whole. The event's opening animation took 56 AI-assisted iterations. AI widened what was possible; human scrutiny made it good.

**4. Unleash creativity and artistry.** Chats, charts and command lines are not the peak of interaction design. "The best practices are yet to be defined." AI can mass-produce monotony, but it is also the greatest creative catalyst we've had. For the *Built to Grow* cover, a human marbler made the variations and AI helped refine colour and line: human judgment, scaled.

## 4. Named tests

| Test | Ask | Used in |
|---|---|---|
| Swap test | Replace the subject with a different one. Does anything else need to change? If not, it's a template. | `no-slop-ui` ("One template for every subject"), `editor-pass` ("Swap test") |
| Green cup | "The cup is green but may as well have been blue." Can you name the reason for each visible decision? | `editor-pass` ("Green-cup audit"), stage 3 questions |
| One level deeper | What need did the user not state but will have? | `editor-pass` ("One level deeper"), stage 1 questions |
| Burrito check | Am I accepting this because it's fast, or because it's good? | stage 3 questions |
| Iterate, don't accept | The first result is the most probable one. What would the next versions try? | stage 3 questions |
| Who maintains this | Who owns it in six months, and what breaks first? | `editor-pass` ("Ownership"), stage 4 questions |

## 5. Tactics and culture

- **Improve the inputs.** Say what you believe, what good is, and what you care about, and attach your source material. "A website for my Korean barbecue" isn't enough.
- **Stress the outputs.** Push one step further, and use adversarial agents to critique the work. You still have to keep pushing yourself.
- **Protect the strange.** Cookie-cutter patterns are safe and cozy. If you lead a team, don't just tell them to use AI; give them room to explore. "AI lowers the cost to create. Let's spend some of that savings on making something truly special." Raise the ceiling, not just the floor.
- What impresses users isn't that something was animated in 30 minutes. It's that their problem was solved, with touches that show someone anticipated their needs.
