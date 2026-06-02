# Reviewer Profiles

Detailed style profiles for each tracked reviewer, derived from analysis of 6,994 review comments.

## David Garcia Mora (`davidgm0`)

**Team:** CX (Customer) | **Role:** Staff Software Engineer
**Comments:** 3,871 | **PRs:** 852 | **Active since:** April 2021

### Style
- **Most prolific reviewer** -- 55% of all comments in the database
- **Technically deep:** 50.1% medium/high technicality (highest on team)
- **Directive-leaning:** 20.9% directives, 13.1% questions
- **Moderate suggestion rate:** 8.9% with code suggestions

### What He Checks For
1. **Testing** (23.4%) -- always expects test coverage
2. **Database** (12.3%) -- queries, migrations, indexes
3. **Naming** (12.0%) -- variable and method names
4. **Error handling** (11.6%) -- rescue blocks, error propagation
5. **Authorization** (8.0%) -- highest on team, reflects CX domain expertise
6. **Security** (2.3%) -- highest security focus

### Blocking Patterns
- 35 blocking comments (0.9% rate, highest absolute count)
- Blocks on: security issues, missing tests, broken authorization

### Communication
- 75.1% conversational, 14.4% detailed
- Average 30.1 words per comment
- Low praise rate (4.3%) -- focuses on substance over encouragement

### How to Use David's Patterns
When reviewing CX-team code or auth-heavy changes, query his patterns:
```bash
$ANALYZER query --db $DB --reviewer davidgm0 --topic authorization --limit 10
$ANALYZER query --db $DB --reviewer davidgm0 --category security --limit 10
```

---

## Roger (`roger-cobalt`)

**Team:** DL (Delivery) | **Role:** Software Engineer
**Comments:** 1,250 | **PRs:** 344 | **Active since:** July 2022

### Style
- **Show don't tell:** 16.6% code suggestion rate (highest on team, 4x Paul's rate)
- **Concise:** 25.0 avg words (shortest on team), 12.1% terse
- **Constructive:** 32.3% constructive sentiment (highest)
- **High technicality variance:** 17.2% high-technicality (highest proportion)

### What He Checks For
1. **Testing** (22.8%) -- strong test advocate
2. **Database** (15.0%) -- schema and query patterns
3. **API design** (10.9%) -- endpoints, serializers, response shapes
4. **Naming** (14.4%) -- clear naming matters
5. **Error handling** (10.2%)

### Communication
- 78.2% conversational, very few detailed comments
- When he has something to say, he says it with code
- Uses suggestions as the primary feedback mechanism

### How to Use Roger's Patterns
When reviewing API/controller code, query his suggestion-heavy style:
```bash
$ANALYZER query --db $DB --reviewer roger-cobalt --topic api-design --limit 10 --verbose
$ANALYZER search --db $DB "serializer response" --reviewer roger-cobalt --limit 5
```

---

## Paul Lucian Ursache (`Lucianolo`)

**Team:** DL (Delivery) | **Role:** Software Engineer
**Comments:** 1,107 | **PRs:** 321 | **Active since:** February 2020 (longest tenure)

### Style
- **Question-driven:** 26.6% questions (nearly 2x team average)
- **Collaborative:** asks "why" rather than "do this"
- **Low technicality:** 81.0% low-technicality -- focuses on patterns, not implementation details
- **Moderate suggestion rate:** 4.2% (prefers discussion over code fixes)

### What He Checks For
1. **Testing** (23.4%) -- consistent with team norm
2. **Database** (16.1%) -- queries and schema
3. **Error handling** (13.7%) -- higher than average
4. **Naming** (12.8%) -- clarity and intent
5. **General** (18.0%) -- broad feedback not tied to a specific domain

### Communication
- 75.2% conversational, 13.9% detailed
- Average 29.9 words per comment
- 4.5% positive sentiment -- more encouraging than Roger or Mauricio
- Questions often start with "why", "what if", "have you considered"

### How to Use Paul's Patterns
When reviewing error handling or database code, query his question style:
```bash
$ANALYZER query --db $DB --reviewer Lucianolo --curiosity question --topic error-handling --limit 10
$ANALYZER query --db $DB --reviewer Lucianolo --topic database --limit 10
```

---

## Mauricio Reis (`mauricio-reis`)

**Team:** DL (Delivery) | **Role:** Software Engineer
**Comments:** 766 | **PRs:** 204 | **Active since:** December 2021

### Style
- **Database specialist:** 19.5% database topic (highest on team)
- **Balanced approach:** even mix of observation, suggestion, directive, question
- **Medium-length communicator:** 42.4% of comments are 21-50 words (most consistent length)
- **Low blocking:** only 1 blocking comment (0.1% rate)

### What He Checks For
1. **Testing** (19.6%) -- slightly below team average but still #1
2. **Database** (19.5%) -- strongest database focus
3. **General** (17.1%)
4. **Naming** (15.9%) -- high naming focus
5. **Error handling** (12.0%)

### Communication
- 82.4% conversational (most conversational on team)
- Average 27.1 words per comment
- 1.0% positive sentiment (lowest praise, highest negative at 10.6%)
- Direct and functional -- points out problems, suggests fixes

### How to Use Mauricio's Patterns
When reviewing database code, migrations, or data layer changes:
```bash
$ANALYZER query --db $DB --reviewer mauricio-reis --topic database --limit 15 --verbose
$ANALYZER query --db $DB --reviewer mauricio-reis --category performance --limit 10
```

---

## Cross-Team Patterns

### What They All Agree On
- **Testing is #1** -- every reviewer's top topic
- **Naming matters** -- all have 12-16% naming focus
- **Conversational tone** -- 75%+ across the board
- **Low blocking** -- <1% blocking rate, reviews are advisory

### Where They Differ
| Dimension | Strongest |
|-----------|-----------|
| Questions | Paul (26.6%) |
| Code suggestions | Roger (16.6%) |
| Technical depth | David (50.1% med+high) |
| Database focus | Mauricio (19.5%) |
| Security focus | David (2.3%) |
| Authorization | David (8.0%) |
| API design | Roger (10.9%) |
| Praise | Paul (4.5%) |
