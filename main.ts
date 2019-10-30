import express, { Request, Response, NextFunction } from 'express';
import bodyParser from 'body-parser';
import axios from 'axios';
import URL from 'url';

import pkg from './package.json';
import config from './config.json';

const app = express();
app.use(bodyParser.json());

app.use((req: Request, res: Response, next: NextFunction) => {
    if(req.method === "OPTIONS"){
        res.status(200);
    }
    
    if(!!req.get('origin')){
        if(config.allowedOrigins.some((allowedOrigin) => {
            let origin = URL.parse(req.get('origin') as string);
            return `${origin.protocol}//${origin.hostname}` === allowedOrigin;
        })) res.header('Access-Control-Allow-Origin', req.get('origin'));
    }
    
    
    res.header('Access-Control-Allow-Headers', '*');
    next();
});

app.get('/', async (req: Request, res: Response) => {
    res.json({});
});

app.post('/', async (req: Request, res: Response) => {
    if(!['method', 'url', 'headers'].every((property) => req.body[property] !== undefined)){
        return res.json({
            success: false,
            data: {
                message: "Invalid request body sent."
            }
        });
    }
    
    const {
        method,
        url,
        auth,
        headers,
        data
    } = req.body;
    
    try {
        const payload = await axios({
            method,
            url,
            auth,
            headers,
            data
        });
        
        return await res.json({
            success: true,
            data: payload.data,
            status: payload.status,
            statusText: payload.statusText,
            headers: payload.headers,
        });
        
    } catch (error) {
        if (error.response) {
            const errorResponse = error.response;
            return res.json({
                success: false,
                data: errorResponse.data,
                status: errorResponse.status,
                statusText: errorResponse.statusText,
                headers: errorResponse.headers,
            });
        } else {
            return res.status(500).send({
                success: false,
                data: {
                    message: "An unknown error occurred whilst the proxy was processing your request."
                }
            });
        }
    }
});

app.listen(config.port, () => console.log(`${pkg.humanName} listening on http://localhost:${config.port}/`));