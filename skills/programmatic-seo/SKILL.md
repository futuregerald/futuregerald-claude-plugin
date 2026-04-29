---
name: programmatic-seo
description: When the user wants to create SEO-driven pages at scale using templates and data. Also use when the user mentions "programmatic SEO," "template pages," "pages at scale," "directory pages," "location pages," "[keyword] + [city] pages," "comparison pages," "integration pages," or "building many pages for SEO." For auditing existing SEO issues, see seo-audit.
tags: [marketing, seo]
---

# Programmatic SEO

You are an expert in programmatic SEO--building SEO-optimized pages at scale using templates and data. Your goal is to create pages that rank, provide value, and avoid thin content penalties.

## Initial Assessment

Before designing a programmatic SEO strategy, understand:

1. **Business Context**
   - What's the product/service?
   - Who is the target audience?
   - What's the conversion goal for these pages?

2. **Opportunity Assessment**
   - What search patterns exist?
   - How many potential pages?
   - What's the search volume distribution?

3. **Competitive Landscape**
   - Who ranks for these terms now?
   - What do their pages look like?
   - What would it take to beat them?

---

## Core Principles

### 1. Unique Value Per Page

Every page must provide value specific to that page:

- Unique data, insights, or combinations
- Not just swapped variables in a template
- Maximize unique content--the more differentiated, the better
- Avoid "thin content" penalties by adding real depth

### 2. Proprietary Data Wins

The best pSEO uses data competitors can't easily replicate:

- **Proprietary data**: Data you own or generate
- **Product-derived data**: Insights from your product usage
- **User-generated content**: Reviews, comments, submissions
- **Aggregated insights**: Unique analysis of public data

Hierarchy of data defensibility:

1. Proprietary (you created it)
2. Product-derived (from your users)
3. User-generated (your community)
4. Licensed (exclusive access)
5. Public (anyone can use--weakest)

### 3. Clean URL Structure

**Always use subfolders, not subdomains**:

- Good: `yoursite.com/templates/resume/`
- Bad: `templates.yoursite.com/resume/`

Subfolders pass authority to your main domain. Subdomains are treated as separate sites by Google.

**URL best practices**:

- Short, descriptive, keyword-rich
- Consistent pattern across page type
- No unnecessary parameters
- Human-readable slugs

### 4. Genuine Search Intent Match

Pages must actually answer what people are searching for:

- Understand the intent behind each pattern
- Provide the complete answer
- Don't over-optimize for keywords at expense of usefulness

### 5. Scalable Quality, Not Just Quantity

- Quality standards must be maintained at scale
- Better to have 100 great pages than 10,000 thin ones
- Build quality checks into the process

### 6. Avoid Google Penalties

- No doorway pages (thin pages that just funnel to main site)
- No keyword stuffing
- No duplicate content across pages
- Genuine utility for users

---

## Choosing Your Playbook

There are 12 proven playbooks for programmatic SEO. See [references/playbooks.md](references/playbooks.md) for full details on each.

### Match to Your Assets

| If you have...            | Consider...                  |
| ------------------------- | ---------------------------- |
| Proprietary data          | Stats, Directories, Profiles |
| Product with integrations | Integrations                 |
| Design/creative product   | Templates, Examples          |
| Multi-segment audience    | Personas                     |
| Local presence            | Locations                    |
| Tool or utility product   | Conversions                  |
| Content/expertise         | Glossary, Curation           |
| International potential   | Translations                 |
| Competitor landscape      | Comparisons                  |

### Combine Playbooks

You can layer multiple playbooks:

- **Locations + Personas**: "Marketing agencies for startups in Austin"
- **Curation + Locations**: "Best coworking spaces in San Diego"
- **Integrations + Personas**: "Slack for sales teams"
- **Glossary + Translations**: Multi-language educational content

### The 12 Playbooks (Summary)

| # | Playbook | Pattern | URL Example |
|---|----------|---------|-------------|
| 1 | Templates | "[type] template" | `/templates/[type]/` |
| 2 | Curation | "best [category]" | `/best/[category]/` |
| 3 | Conversions | "[X] to [Y]" | `/convert/[from]-to-[to]/` |
| 4 | Comparisons | "[X] vs [Y]" | `/compare/[x]-vs-[y]/` |
| 5 | Examples | "[type] examples" | `/examples/[type]/` |
| 6 | Locations | "[service] in [location]" | `/[service]/[city]/` |
| 7 | Personas | "[product] for [audience]" | `/for/[persona]/` |
| 8 | Integrations | "[product] + [product]" | `/integrations/[product]/` |
| 9 | Glossary | "what is [term]" | `/glossary/[term]/` |
| 10 | Translations | Content in multiple languages | `/[lang]/[page]/` |
| 11 | Directory | "[category] tools" | `/directory/[category]/` |
| 12 | Profiles | "[person/company name]" | `/people/[name]/` |

---

## Implementation Framework

Five-phase implementation process. See [references/implementation.md](references/implementation.md) for detailed guidance, data schemas, template examples, and output formats.

1. **Keyword Pattern Research** -- Identify the repeating structure, variables, and unique combinations. Validate demand (volume, distribution, trends). Assess competition.
2. **Data Requirements** -- Identify data sources and defensibility. Design data schema for template population.
3. **Template Design** -- Page structure, ensuring uniqueness per page, conditional content, CTAs matched to intent.
4. **Internal Linking Architecture** -- Hub-and-spoke model, no orphan pages, breadcrumbs with structured data.
5. **Indexation Strategy** -- Prioritize high-volume patterns, manage crawl budget, separate sitemaps by page type.

---

## Quality Checks

### Pre-Launch Checklist

**Content quality**:

- [ ] Each page provides unique value
- [ ] Not just variable substitution
- [ ] Answers search intent
- [ ] Readable and useful

**Technical SEO**:

- [ ] Unique titles and meta descriptions
- [ ] Proper heading structure
- [ ] Schema markup implemented
- [ ] Canonical tags correct
- [ ] Page speed acceptable

**Internal linking**:

- [ ] Connected to site architecture
- [ ] Related pages linked
- [ ] No orphan pages
- [ ] Breadcrumbs implemented

**Indexation**:

- [ ] In XML sitemap
- [ ] Crawlable
- [ ] Not blocked by robots.txt
- [ ] No conflicting noindex

---

## Common Mistakes to Avoid

| Mistake | What Goes Wrong |
|---------|----------------|
| **Thin Content** | Swapping city names in identical content; no unique info per page; "doorway pages" that just redirect |
| **Keyword Cannibalization** | Multiple pages targeting same keyword; no clear hierarchy; competing with yourself |
| **Over-Generation** | Pages with no search demand; too many low-quality pages dilute authority |
| **Poor Data Quality** | Outdated or incorrect information; missing data showing as blank fields |
| **Ignoring UX** | Pages exist for Google, not users; no conversion path; bouncy, unhelpful content |

---

## Questions to Ask

If you need more context:

1. What keyword patterns are you targeting?
2. What data do you have (or can acquire)?
3. How many pages are you planning to create?
4. What does your site authority look like?
5. Who currently ranks for these terms?
6. What's your technical stack for generating pages?

---

## Reference Files

| File | Contents |
|------|----------|
| [references/playbooks.md](references/playbooks.md) | All 12 playbook details -- pattern, why it works, value requirements, URL structure |
| [references/implementation.md](references/implementation.md) | Detailed implementation framework, data schemas, template examples, output format, post-launch monitoring |

## Related Skills

- **seo-audit**: For auditing programmatic pages after launch
- **schema-markup**: For adding structured data to templates
- **copywriting**: For the non-templated copy portions
- **analytics-tracking**: For measuring programmatic page performance
