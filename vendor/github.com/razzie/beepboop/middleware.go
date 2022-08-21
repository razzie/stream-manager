package beepboop

// Middleware is a function called before a page handler
// to be able to change or intercept the request
type Middleware func(pr *PageRequest) *View
