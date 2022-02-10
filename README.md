# Creating your own custom tags

## Introduction
> NOTE: You can [skip](#Retrieving tags) this if you already know what a struct tag is/does

In this article we will have a look at the Go feature "struct tags", talk about what they are and how they are used. Furthermore, we will explore how we can create our own custom tags, to benefit even more from this useful feature.

So, to start with: What are struct tags? Struct tags are a Go feature, which defines a tag for a particular field of a struct. In itself, a struct tag is merely text metadata field. However, it's quite practical and is commonly used for defining behaviour for a particular field. A very common example of this is the `json` tag, use by the `json` package to marshal and unmarshal struct fields:

```go
type Person struct {
	Email string `json:"email"`
}
```

The above example instructs the `json` package to write the `Person::Email` field as `email` when encoding to JSON, rather than `Email`, as well as decoding the value of `email` field in a JSON and assigning it to `Person::Email`. Simple! This is really useful, because it's a very concise way of specifying how to convert from and to JSON. This is the essence of struct tags: Doing something useful, in a concise manner. There are many use-cases for struct tags, other than defining variable naming convention conversion (say that 5 times in a row quickly :flushed:). This article will explain how to create your own custom tag and (hopefully) save yourself from writing the same code over and over again!

## Retrieving tags
The first hurdle we need to tackle, is figuring out how we can retrieve these tags programmatically. Fortunately,
retrieving a tag, is pretty straight forward, using the `reflect` package. As with any article which mentions the `reflect` package, a warning must ensue. The `reflect` package is a powerful package and gives Go developers the flexibility to create some very useful and creative projects. However, one must proceed with caution! The `reflect` package is unforgiving and errors are typically handled with a `panic`. We will see examples of this later in the article.

However, with that out of the way, here is how to retrieve struct tags using the `reflect` package:

```go
func PrintTags(v interface{}) {
	val := reflect.ValueOf(v)
	kind := val.Kind()
	switch kind {
	case reflect.Struct:
		typ := val.Type()
		for i := 0; i < typ.NumField(); i++ {
			fmt.Println(typ.Field(i).Tag)
		}
		return
	}
}
```

In the above function, we have created a function which accepts an `interface{}` value (in other words, this can be *any* value). Working with the `interface{}` type in Go, is a little underwhelming. It has no methods nor fields and there is generally very little which you can actually do with an `interface{}` other than type asserting. So, we would therefore like some more information on what this `interface{}` value actually contains. For this we use the `reflect.ValueOf` function. This returns a `reflect.Value`, which contains the metadata we need to work with our given value:

```go
// From 'value.go' in the reflect package
type Value struct {
	typ *rtype
	ptr unsafe.Pointer
	flag
}
```

Very simply put, the `Value` struct is a structure containing various metadata and pointers for a given variable. More so, it has various methods attached, to enable retrieving this information in a, somewhat, safe and easy manner. In any case, it's better than retrieving the data using pointer arithmetic, which is what most of the methods are doing under the hood. The three fields of Value are:
* `typ`: `*rtype` is a struct for generically describing *any* value. Which includes type and kind name, as well as metadata establishing size, hashing, equality, as well as information on garbage collection. 
* `ptr`: is an `unsafe.Pointer` (which is as close to a C pointer as Go gets), to the data stored by the given value.
* `flag`: is another metadata field, which is typically used for pointer arithmetic.

I'm not going deeper into this rabbit hole, but if are curious, I can highly recommend jumping right in! It's a lot of fun and you learn a lot about how Go works under the hood. So, if you're into that kind of stuff, this is your gateway.

> NOTE : Thoroughly recommend this article series: https://cmc.gitbook.io/go-internals/

Either way, the `reflect.Value` type allows us to have a peak at the metadata of the given value. For example, using the method `reflect.Value::Kind` we can retrieve the underlying 'kind' (int, array, slice, struct etc.). Using this kind value, we can check whether the given value is of type `reflect.Struct`. We do this, as we are not interested in anything else; After all, we are trying to retrieve struct tags, and they only reside on structs.

> NOTE: I will try to distinguish between type and kind throughout the article, in terms of what it means according to the reflect package. The difference being: `kind` can be considered the native / primitive types of Go: `struct`, `int(s)`, `string`, `float` etc. Whereas `type` also includes our custom structs such as `reflect.Value`. In other words: `reflect.Value`'s kind is a `Struct`, but is of type `reflect.Value`. A pointer, so `*reflect.Value`'s kind is `Ptr`. 

Should we have been lucky enough to receive a struct kind, we will now retrieve the type information of this struct. As an example, if we had received a `Person` type, we would be retrieving the `reflect.Type` metadata for a `Person`. With this type information we can now iterate over the fields by calling the `reflect.Type::NumField` method, which will return the number of fields for that type. Thereafter, we can retrieve the metadata for each field using the method `reflect.Value::Field`, specifying the field index with our iterator `i`.

Last, but not least, we can now access the `reflect.Field::Tag` property, which is indeed the struct tag for that particular field. So, let's take it for a spin:


```go
type Person struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

func main() {
	PrintTags(Person{
		FirstName: "Lasse Martin",
		LastName:  "Jakobsen",
		Email:     "lasse@tengen.dk",
	})
}

> go run main.go
json:"first_name"
json:"last_name"
json:"email"
```

This is great! We can already feel the power of the `reflect` package >:) Our newly created `Person` type has three fields with tags, which are all being printed as expected. However, there is still work to be done!

```go
type Person struct {
	Name      Name   `json:"name"`
	Email     string `json:"email"`
}

type Name struct {
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
}

func main() {
	PrintTags(Person{
		Name: Name{
			FirstName: "Lasse Martin",
			LastName:  "Jakobsen",
		},
		Email:     "lasse@tengen.dk",
	})
}

> go run main.go
json:"name"
json:"email"
```

Only the tags `name` and `email` are being printed and the tags `first_name` and `last_name` are being ignored. This is because we have moved first and last name fields into a struct of their own `Name`. Our current functionality only looks at the struct tags of the given struct, but does not consider that one of the fields could be a struct itself, with a set tags of it's own. We there need to recursively check our fields, if they themselves, contain struct tags:

```go
func PrintTags(v interface{}) {
	val := reflect.ValueOf(v)
	kind := val.Kind()
	switch kind {
	case reflect.Struct:
		handleStruct(val)
	}
}

func handleStruct(val reflect.Value) {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		field := typ.Field(i)
		fmt.Println(field.Tag)
		switch val.Field(i).Kind() {
		case reflect.Struct:
			handleStruct(val.Field(i))
		}
	}
	return
}
```

We have simply moved our logic into another function `handleStruct` which in turn checks if one of the fields of the given struct, is a struct itself. If so, then we simply call `handleStruct` again. Easy peasy! Running our `main` function now, will yield all of the tags of the inner struct :thumbs_up:

However, we also need to think about other kinds than inner structs; We also need to think about structs containing arrays, maps etc, which in turn, could also contain structs. However, this should be fairly simple to handle, as we can just add a few more handlers for our various types.

```go
func handleValue(val reflect.Value) {
	kind := val.Kind()
	switch kind {
	case reflect.Struct:
		handleStruct(val)
	case reflect.Array, reflect.Slice:
		handleArray(val)
	case reflect.Map:
		handleMap(val)
	case reflect.Ptr:
		handleValue(val.Elem())
	}
}

func handleStruct(val reflect.Value) {
	typ := val.Type()
	for i := 0; i < typ.NumField(); i++ {
		handleValue(val.Field(i))
	}
	return
}

func handleArray(val reflect.Value) {
	for i := 0; i < val.Len(); i++ {
		handleValue(val.Index(i))
	}
}

func handleMap(val reflect.Value) {
	for _, key := range val.MapKeys() {
		handleValue(val.MapIndex(key))
	}
}
```

Great! We have added three new handlers, so that we are now handling structs, arrays/slices and maps. Each of them have a slightly difference syntax for iterating through their contents. For arrays and slices, we are using the `reflect.Value::Len` method to retrieve the length of the array and `reflect.Value::Index` for retrieving the element at the specified index. For maps we are iterating through the keys of the map and retrieving the value stored for that particular key.

It's important to note, that the `reflect.Value::NumField` and `reflect.Value::MapKeys` methods are specific to, respectively, structs and maps. If these methods are called on a different value kind, it will cause a panic, which we want to avoid at all costs.

Not to forget, we have also added `handleValue` which acts as a distributor, identifying the kind of the value and invoking the corresponding function for that kind.

> NOTE : We have also added a handler for pointers in `handleValue`. This is because `reflect` will identify a pointer as a `reflect.Ptr` rather than a struct (which makes sense). So, calling the `.Elem()`, essentially is the same as de-referencing, returning the value of that pointer.

## Creating Custom Tags
### The building blocks
So far so good, but currently we are only accessing the field tag and printing them, which is lovely, but not particularly useful. So, let's set out to do something useful. To prepare for this, let's do a super quick refactor of our code, to make our lives a little easier in the not so distant future:

```go
type TagHandler struct {
	HandlerFn func(value reflect.Value, field reflect.StructField) error
}

func (th TagHandler) Handle(v interface{}) error {
	return th.handleValue(reflect.ValueOf(v))
}


func (th TagHandler) handleStruct(val reflect.Value) error {
	typ := val.Type()
	for i := 0; i < val.NumField(); i++ {
		if err := th.HandlerFn(val.Field(i), typ.Field(i)); err != nil {
			return err
		}
		if err := th.handleValue(val.Field(i)); err != nil {
			return err
		}
	}
	return nil
}

func (th TagHandler) handleValue(val reflect.Value) error { ... }

func (th TagHandler) handleArray(val reflect.Value) error { ... }

func (th TagHandler) handleMap(val reflect.Value) error { ... }
```

We have created a new structure `TagHandler` and have made all of our functions into methods of this struct. Furthermore, `TagHandler` stores a function with the signature `func(reflect.Value, reflect.StructField) error`, the idea behind this, is to allow any function with this signature to be called by the `TagHandler::handleStruct` method. This enables us to, very easily, create functionality for our custom tags. So let's try it out!

### Regex Validator Tag
To start off with, we are going to create a custom tag which will be able to validate the value of a tagged field, using a regular expression.

```go
func handleValidateTag(value reflect.Value, field reflect.StructField) error {
	tag, ok := field.Tag.Lookup("validate")
	if !ok {
		return nil
	}
	match, err := regexp.Compile(tag)
	if err != nil {
		return fmt.Errorf("validation regexp syntax error: %v", err)
	}
	if !match.MatchString(value.String()) {
		return fmt.Errorf("invalid field (%v::%v) %v != %v", field.Type, field.Name, value.String(), tag)
	}
	return nil
}
```

The function `handleValidateTag` receives a `reflect.Value` and a `reflect.StructField`. Using the struct field, we lookup the value for the `validate` tag. If it doesn't exist (`ok` returns as false), then we know that there is no `validate` tag and therefore nothing to validate, so we can safely just return. However, if there is a tag, we attempt to compile it and then match the field value with our tag regular expression. If there is no match, then the value is considered invalid, so we return an error. If there is a match, we can assume that the value is valid. Let's try it out!

```go
type Person struct {
	...
	Email string `json:"email" validate:"^[\\w-\\.]+@([\\w-]+\\.)+[\\w-]{2,4}$"`
	...
}

func main() {
	th := TagHandler{
		HandlerFn: handleValidateTag,
	}

	err := th.Handle(Person{
		Name: Name{
			FirstName: "Lasse Martin",
			LastName:  "Jakobsen",
		},
		Email:     "lasse@tengen.dk",
		Friends: []*Person{
			{
				Name: Name{ FirstName: "Iaf", LastName: "Nofrens"},
				Email: "l33tboi95@hotmail",
			},
		},
	})
	fmt.Println(err)
}
```

> NOTE : The regex value for validating an e-mail is not perfect, but it should suffice for the purposes for this article :relaxed_smile:

Notice that this will return an error, because the e-mail in the friends slice is invalid. If we fix this email address by giving it a `.com` postfix, the error is resolved ! Magic ! :party:

Of course, our `handleValidateTag` is still a rather naive function. For example, it assumes that all fields will be of string value. This is an issue! It is easily imaginable that we wanted to validate something else, such as an integer. Let's try to add a `BirthYear` integer field to our `Person` type and see what happens, when we run our program.

```go
type Person struct {
	BirthYear int `json:"birth_year" validate:"^(19|20)\\d\\d$"`
	...
}

Output:
invalid field (int::BirthYear) <int Value> != ^(19|20)\d\d$
```

So, this is because of the following line of code:
```go
func handleValidateTag(value reflect.Value, field reflect.StructField) error {
	...
	if !match.MatchString(value.String()) { ... }
	...
}
```

We are trying to access the string value of our `reflect.Value` using the method `reflect.Value::String`. However, in this case, our underlying value is actually an integer, so `reflect` returns the string value `<int Value>`. So, thankfully not a panic, but nevertheless, completely useless. We will handle this lazily, but effectively but converting our type to string with `fmt.Sprintf` rather than using `reflect.Value::String`

```go
func valueToString(value reflect.Value) string {
	return fmt.Sprintf("%v", value.Interface())
}

func handleValidateTag(value reflect.Value, field reflect.StructField) error {
	...
	str := valueToString(value)
	if !match.MatchString(str) { ... }
	...
}
```

We have created a new function `valueToString` which uses `fmt.Sprintf` to return a string from the underlying `interface{}` contained in the `reflect.Value`. This is probably not the most efficient way of doing this, but it certainly does the job. If we run our program again, we will get the following output:

```
invalid field (int::BirthYear) 0 != ^(19|20)\d\d$
```

And if we set the value of `BirthYear` values (remember the `Person` in `Friends`) to something valid (within this century), our validator will stop complaining :party: There are of course many other cases we are not accounting for, but for now, we will put our validator on the shelf and move on to something else.

### Config Tag
So, now that we have seen that we can validate our struct field values via. our tags, how about we have a look at using struct tags for *setting* the values of our struct fields? Let's try to make a struct tag, in which we can specify the environment variable which should populate the value of our config parameter. This is a pretty common use-case and something that has been done many times before, but let's try doing this ourselves, to see what it involves.

Firstly, let's have a look at the syntax to use for specifying our environment variable parameters. I suggest that we start of simple, specifying only the name of our environment variable holding the value, so our config struct would look something like the following:

```go
type Config struct {
	HttpMaxRetries    int    `conf:"HTTP_MAX_RETRIES"`
	ElasticsearchHost string `conf:"ELASTICSEARCH_HOST"`
}
```

Now it's time to create our handler for reading the environment variables and setting the retrieved value for the tagged field:

```go
func handleConfigTag(value reflect.Value, field reflect.StructField) error {
	tag, ok := field.Tag.Lookup("conf")
	if !ok {
		return nil
	}
	envvar, ok := os.LookupEnv(tag)
	if !ok {
		return nil
	}
	return setValue(value, envvar)
}

func setValue(value reflect.Value, envvar string) error {
	switch value.Kind() {
	case reflect.String:
		value.SetString(envvar)
	case reflect.Int:
		n, err := strconv.Atoi(envvar)
		if err != nil {
			return err
		}
		value.SetInt(int64(n))
	}
	return nil
}
```

As we did with our validation handler, we start by retrieving the tag (in this case "conf"). After this, we then try to retrieve the environment variable specified in the tag. As we do with the tag, if there is no value, we simply return immediately and assume there is nothing to be done for this environment variable. If we do retrieve a value, we then set the value of our field, using the `setValue` function.

In this function, we start by identifying the `reflect.Kind` of the field value and attempt to convert the environment variable string to the appropriate type. If the kind is a string, we can simple set the value using `reflect.Value::SetString`, but if our field is a `reflect.Int`, we will attempt to convert the environment variable string value to an integer and thereafter set our field value using the `reflect.Value::SetInt` method.

Currently, we are merely supporting configuration types of `int` and `string`, but it won't take much to add support for other types. If we wanted to, we could go as far as adding support for slices, structs etc. ... However, we won't go that far in this article :sweat_smile:

> NOTE: Furthermore, you could also add more parameters and specify default values and usage messages.

Instead, let us test out our simple new configuration, to see if it works!

```go
func main() {
	cfgHandler := TagHandler{
		HandlerFn: handleConfigTag,
	}

	var cfg Config
	err = cfgHandler.Handle(&cfg)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Printf(`ElasticsearchHost: %s, HttpMaxRetries: %d\n`,
		cfg.ElasticsearchHost, cfg.HttpMaxRetries)
}
```
As in our previous program, we initialise our `TagHandler` by passing the `handleConfigTag` as the internal `HandlerFn` to create our custom tag behaviour, for setting struct fields through environment variables. We then declare a `cfg` variable and pass a pointer of this variable to our `TagHandler::Handle` method. It's important that we pass a pointer, rather than a copy, to ensure that when we set the various configuration field values, we are setting the values of our original `cfg` variable, rather than on a copy. This is the exact same mechanism behind the function `json.Unmarshal`.

Finally, we print the values of our `cfg` variable to ensure that our handler is working as intended. Running the program yields the following results:

```bash
> ELASTICSEARCH_HOST=http://localhost:9200 HTTP_MAX_RETRIES=5 go run main.go
ElasticsearchHost: http://localhost:9200, HttpMaxRetries: 5
```

Great success!

As said before, this is a rather simple implementation and there would still be a long way to go, before this would be of actual use. We would have to support all the various types (int32, int64, float types, structs, arrays, etc.) or figure out some abstraction to simplify our approach. However, I hope that the examples served their purpose as a quick introduction.

## Summary
In this article we covered the definition and usage of struct tags, as well as how to create our own custom tags. Of course, the examples in the article were simple (and incomplete) solutions, but I hope they demonstrated the building blocks for building your own custom struct tags. As mentioned before, the `reflect` package gives developers a lot of flexibility and therefore the possibilities are technically endless. If you really wanted to, it would be possible to write your own scripting language and evaluate this in your custom tag handler execution... However, let's just say, this is an idea beyond terrible.

That being said, you can have endless amount of fun with the `reflect` package. Even if it's not for anything useful, sometimes it's just a lot of fun to experiment and try out some whacky experiments. Most of the things I've learnt, have come from side-projects and whacky experimentation, which lead absolutely nowhere :joy:

I hope this article was useful! If you have questions or requests, then please feel free to reach out to me: lasse@jakobsen.dev - If you enjoyed the article, then be sure to have a look at https://jakobsen.dev for more of my articles.

Thanks! :bow: