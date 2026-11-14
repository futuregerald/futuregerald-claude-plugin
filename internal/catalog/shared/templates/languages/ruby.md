## Ruby/Rails Rules

### Style

- Follow Ruby Style Guide conventions
- Use 2-space indentation
- Prefer `&&`/`||` over `and`/`or`
- Use `snake_case` for methods and variables, `CamelCase` for classes

```ruby
# Before
class userController < ApplicationController
  def getUser
    user = User.find_by_id(params[:id])
    if user == nil
      render json: { error: "Not found" }, status: 404
    else
      render json: user
    end
  end
end

# After
class UsersController < ApplicationController
  def show
    user = User.find_by(id: params[:id])
    return render json: { error: "Not found" }, status: :not_found unless user

    render json: user
  end
end
```

### Best Practices

- Use strong parameters for mass assignment protection
- Prefer scopes over class methods for queries
- Use `present?`/`blank?` over `nil?` checks when appropriate
- Keep controllers thin, models fat (but not too fat)
- Extract service objects for complex business logic

### Rails Conventions

- Follow RESTful routing
- Use concerns for shared model/controller logic
- Prefer `find_by` over `where(...).first`
- Use transactions for multi-step database operations

```ruby
# Service object pattern
class UserRegistrationService
  def initialize(params)
    @params = params
  end

  def call
    ActiveRecord::Base.transaction do
      user = User.create!(@params)
      UserMailer.welcome(user).deliver_later
      user
    end
  rescue ActiveRecord::RecordInvalid => e
    OpenStruct.new(success: false, errors: e.record.errors)
  end
end
```

### Testing

```bash
bundle exec rspec              # Run all tests
bundle exec rspec spec/models  # Run model tests
bundle exec rubocop            # Linting
bundle exec rails db:test:prepare  # Prepare test database
```
