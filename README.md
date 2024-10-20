# pg-migrate
A lightweight CLI database migration tool for Postgres.

### Motivation
When I was working on a realtime chat application, I realized that I needed a database migration tool to make schema updates easier. Although there are many great tools developed by great minds in the developer community, I found the tools too complex
for the project I was working on. For my project, I only needed three features from a migration tool: upstream migrations, downstream migrations, and migrations sql file creation. Tools such as [golang-migrate](https://github.com/golang-migrate/migrate)
and [tern](https://github.com/jackc/tern) offer great features and support variety of DBEngines. I only needed a fraction of the features and only support for postgres. So I decided to build my own migration tool. It turned out to be a great learning experience.


### Installation
Clone the repo and install using go cli
```bash
  git clone https://github.com/Nafis-Anjoom/pg-migrate.git
  cd pg-migrate
  go install
```

### Usage
1. **Initialize migrations with connection string env variable** <br>
   ```bash
     pg-migrate init -database DB_URL
   ```
   The tool will create a config file and migrations directory in the working directory. The migrations directory and the connection string env variable can be modified in the config file.<br>
2. **Create a migration** <br>
   ```bash
     pg-migrate create user_table
   ```
   The tool will create an upstream and downstream migration file with incrementing version number and provided name. <br>
3. **Migrate** <br>
   Update to the latest version:
   ```bash
     pg-migrate migrate
   ```
   Migrate to a specific version:
   ```bash
     pg-migrate migrate -target 5
   ```
   Migrate incrementally:
   ```bash
     pg-migrate migrate -target +5
     pg-migrate migrate -target -5
   ```
   Drop the schema:
   ```bash
     pg-migrate migrate -target 0
   ```

### Future plans
1. Improve performance by prefetching some migration files
2. Refactor code to handle errors elegantly
3. Potentially improve performance by using a more lightweight driver
4. Come up with a compression algorithm to potentially store all the migration files in a database efficiently
