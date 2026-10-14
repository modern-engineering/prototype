package main

func Example() {
	main()
	// Output:
	// database: job.Descriptor, can instantiate: true
	// backend: application.Descriptor, can instantiate: true
	// placement: catalogue.Placement, can instantiate: false
	// database.region = "eu-west"
	// backend.database <- database.address
}
