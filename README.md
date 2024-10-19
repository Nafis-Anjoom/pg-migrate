# pg-migrate
A lightweight CLI database migration tool for Postgres.

### Motivation
When I was working on a realtime chat application, I realized that I needed a database migration tool to make schema updates easier. Although there are many great tools developed by great minds in the developer community, I found the tools too complex
for the project I was working on. For my project, I only needed three features from a migration tool: upstream migrations, downstream migrations, and migrations sql file creation. Tools such as [golang-migrate](https://github.com/golang-migrate/migrate)
and [tern](https://github.com/jackc/tern) offer great features and support variety of DBEngines. I only needed a fraction of the features and only support for postgres. So I decided to build my own migration tool. It turned out to be a great learning experience.
