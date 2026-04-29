# Programmatic SEO -- Implementation Details

Detailed implementation framework, data schemas, templates, output format, and post-launch monitoring. See [../SKILL.md](../SKILL.md) for the high-level overview and core principles.

---

## 1. Keyword Pattern Research

**Identify the pattern**:

- What's the repeating structure?
- What are the variables?
- How many unique combinations exist?

**Validate demand**:

- Aggregate search volume for pattern
- Volume distribution (head vs. long tail)
- Seasonal patterns
- Trend direction

**Assess competition**:

- Who ranks currently?
- What's their content quality?
- What's their domain authority?
- Can you realistically compete?

---

## 2. Data Requirements

**Identify data sources**:

- What data populates each page?
- Where does that data come from?
- Is it first-party, scraped, licensed, public?
- How is it updated?

**Data schema design**:

```
For "[Service] in [City]" pages:

city:
  - name
  - population
  - relevant_stats

service:
  - name
  - description
  - typical_pricing

local_providers:
  - name
  - rating
  - reviews_count
  - specialty

local_data:
  - regulations
  - average_prices
  - market_size
```

---

## 3. Template Design

**Page structure**:

- Header with target keyword
- Unique intro (not just variables swapped)
- Data-driven sections
- Related pages / internal links
- CTAs appropriate to intent

**Ensuring uniqueness**:

- Each page needs unique value
- Conditional content based on data
- User-generated content where possible
- Original insights/analysis per page

**Template example**:

```
H1: [Service] in [City]: [Year] Guide

Intro: [Dynamic paragraph using city stats + service context]

Section 1: Why [City] for [Service]
[City-specific data and insights]

Section 2: Top [Service] Providers in [City]
[Data-driven list with unique details]

Section 3: Pricing for [Service] in [City]
[Local pricing data if available]

Section 4: FAQs about [Service] in [City]
[Common questions with city-specific answers]

Related: [Service] in [Nearby Cities]
```

---

## 4. Internal Linking Architecture

**Hub and spoke model**:

- Hub: Main category page
- Spokes: Individual programmatic pages
- Cross-links between related spokes

**Avoid orphan pages**:

- Every page reachable from main site
- Logical category structure
- XML sitemap for all pages

**Breadcrumbs**:

- Show hierarchy
- Structured data markup
- User navigation aid

---

## 5. Indexation Strategy

**Prioritize important pages**:

- Not all pages need to be indexed
- Index high-volume patterns
- Noindex very thin variations

**Crawl budget management**:

- Paginate thoughtfully
- Avoid infinite crawl traps
- Use robots.txt wisely

**Sitemap strategy**:

- Separate sitemaps by page type
- Monitor indexation rate
- Prioritize by importance

---

## Monitoring Post-Launch

**Track**:

- Indexation rate
- Rankings by page pattern
- Traffic by page pattern
- Engagement metrics
- Conversion rate

**Watch for**:

- Thin content warnings in Search Console
- Ranking drops
- Manual actions
- Crawl errors

---

## Output Format

### Strategy Document

**Opportunity Analysis**:

- Keyword pattern identified
- Search volume estimates
- Competition assessment
- Feasibility rating

**Implementation Plan**:

- Data requirements and sources
- Template structure
- Number of pages (phases)
- Internal linking plan
- Technical requirements

**Content Guidelines**:

- What makes each page unique
- Quality standards
- Update frequency

### Page Template

**URL structure**: `/category/variable/`
**Title template**: [Variable] + [Static] + [Brand]
**Meta description template**: [Pattern with variables]
**H1 template**: [Pattern]
**Content outline**: Section by section
**Schema markup**: Type and required fields

### Launch Checklist

Specific pre-launch checks for this implementation
