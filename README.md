
# FloralAI_Backend

This document provides an overview and documentation for the backend app developed in Go.




![App Screenshot](https://res.cloudinary.com/dqe4ld4cx/image/upload/v1710735729/FloralAI_xwvj43.png)



## Installation

Install FloralAI_Backend using the following command

```bash
 git clone https://github.com/imumar07/FloralAI_Backend.git
```

Install dependencies

```bash
 go mod download
```
    
## Configurations
- Configuration parameters are stored in config.yaml.
- Modify config.yaml as per your environment and requirements.

## Usage/Examples
1. Run the Application
```bash
go run server.go
```
2. The application will start and listen for incoming requests.
Endpoints
- `GET /`: Test endpoint to verify if the application is running.
- `POST /register`: Register a new user.
- `POST /send-data`: Authenticate user and receive data.
- `POST /reset-password`: Reset user password.
- `POST /forget-password`: Initiate password recovery.
- `POST /upload`: Upload an image of a flower and receive detailed information about it.




## Database

- MySQL is used as the database.
- Database schema and migration scripts are provided in the `db` directory.
- Configure database connection details in config.yaml.
## Deployment

- Deploy the application to your preferred hosting environment.
- Set appropriate environment variables for production settings.


## Contributing

Contributions are always welcome!



## License

This project is licensed under the [MIT](https://choosealicense.com/licenses/mit/)

```
https://github.com/imumar07/FloralAI_Backend.git 
Feel free to customize the content further based on your specific application details and requirements.

```

