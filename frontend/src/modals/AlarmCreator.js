import React, { useState, useRef } from 'react';
import { Modal, ModalHeader, ModalBody } from 'reactstrap';
import Cron from 'react-cron-generator';

import axiosObject from '../requests/API';
import { formatDateToLocalISOString } from '../crons';

export const ModalAlarmCreator = ({ toggle, show, setEvents }) => {

    const [msg, setMsg] = useState("");
    const [title, setTitle] = useState("");

    const [timestamps, setTimestamps] = useState([{ date: "", time: "" }]);

    const addTimestamp = () => {
        setTimestamps([...timestamps, { date: "", time: "" }]);
    };

    const deleteTimestamp = (index) => {
        const newTimestamps = [...timestamps];
        newTimestamps.splice(index, 1);
        setTimestamps(newTimestamps);
    };

    const handleTimestampsUpdate = (index, type, event) => {
        const newValue = event.target.value;
        const newTimestamps = [...timestamps];
        newTimestamps[index][type] = newValue;
        setTimestamps(newTimestamps);
    };

    const [crons, setCrons] = useState([]);

    const addCron = () => {
        setCrons([...crons, { cronExpression: "0 0 00 1/1 * ? *", cronDescription: "-" }])
    };

    const deleteCron = (index) => {
        const newCrons = [...crons];
        newCrons.splice(index, 1);
        setCrons(newCrons);
    }

    const updateCron = (index, expr, desc) => {
        const newValue = { cronExpression: expr, cronDescription: desc};
        const newCrons = [...crons];
        newCrons[index] = newValue;
        setCrons(newCrons);
    };

    const handleSubmit = async (event) => {
        event.preventDefault();
        try {
            const dateExpressions = timestamps.map(item => {
                const { date, time } = item;
                if (date === "" || time === "") throw new Error("invalid date");

                const dateObject = new Date(date);
                const [ hours, minutes ] = time.split(":");
                dateObject.setHours(hours);
                dateObject.setMinutes(minutes);

                return formatDateToLocalISOString(dateObject);
            });

            const cronExpressions  = crons.map(item => item.cronExpression);

            const response = await axiosObject.post("/alarms", {
                message: title,
                dates: dateExpressions, 
                crons: cronExpressions,
                timezone: Intl.DateTimeFormat().resolvedOptions().timeZone,
            });
            
            setEvents(events => {
                const newEvents = [...events];
                newEvents.push(response.data);
                return newEvents;
            });

            setMsg("Alarm created!");
            setTimeout(() => {
                setMsg("");
            }, 2500);

        } catch(err) {
            setMsg(err.message);
            setTimeout(() => {
                setMsg("");
            }, 2500);
        }
    };

    return (
        <Modal tabIndex="-1" role="dialog" isOpen={ show } toggle={ toggle }>
            <div role="document">
                <ModalHeader toggle={ toggle } className="bg-dark text-primary text-center">
                    Create Alarm
                </ModalHeader>
                <ModalBody>
                    <div>
                        {msg!==""?<h5 className="mb-4 text-danger">{ msg }</h5>:null}
                        
                        <form onSubmit={handleSubmit}>
                            <div className="form-group">
                                <label>Title</label>
                                <input type="text" className="form-control" value={title} onChange={e => { setTitle(e.target.value) }} />
                            </div>

                            {timestamps.map((item, index) => (
                                <div key={index}>
                                    <div className="form-row">
                                        <div className="col">
                                            <label>Date</label>
                                            <input type="date" className="form-control" value={item.date} onChange={(e) => handleTimestampsUpdate(index, 'date', e)} />
                                        </div>
                                        <div className="col">
                                            <label>Time</label>
                                            <input type="time" className="form-control" value={item.time} onChange={(e) => handleTimestampsUpdate(index, 'time', e)} />
                                        </div>
                                        <div className="col-auto d-flex align-items-end">
                                            <button type="button" className="btn btn-danger mt-4" onClick={() => deleteTimestamp(index)}>Delete</button>
                                        </div>
                                    </div>
                                </div>
                            ))}

                            {crons.map((item, index) => (
                                <CronCreator key={index} index={index} item={item} deleteCron={deleteCron} updateCron={updateCron} />
                            ))}
                            <div className="row">
                                <button type="button" className="btn btn-primary mt-3 col m-2" onClick={addTimestamp}>Add Timestamp</button>
                                <button type="button" className="btn btn-primary mt-3 col m-2" onClick={addCron}>Add Cron</button>
                            </div>
                            <div className="row">
                                <button type="submit" className="btn btn-success mt-3 col m-2">Create Alarm</button>
                            </div>
                        </form>
                    </div>
                </ModalBody>
            </div>
        </Modal>
    );
};

export const CronCreator = ({ index, item, deleteCron, updateCron}) => {

    const cron = useRef(null);

    const [cronResult, setCronResult] = useState(null);

    const toggleCron = () => {
        if (cron.current.classList.contains("show")) {
            cron.current.classList.remove("show");
            return;
        }
       cron.current.classList.add("show");
    };

    const apply = (e) => {
        e.preventDefault();
        updateCron(index, cronResult.expr, cronResult.desc);
        toggleCron();
    };

    return (
        <div key={index}>
            <div className="form-row mt-3">
                <div className="col">
                    <button className="btn btn-primary w-100" onClick={toggleCron} type="button">{item.cronDescription}</button>
                </div>
                <div className="col-auto d-flex align-items-end">
                    <button className="btn btn-danger col-auto" onClick={() => deleteCron(index)}>Delete</button>
                </div>
            </div>
            <div className="form-row">
                <div className="collapse col mt-2" ref={cron}>
                    <Cron showResultText={true} onChange={(expr, desc)=> {setCronResult({expr, desc})}} />
                    <button className="btn btn-success col w-100" onClick={apply}>Apply</button>
                </div>
            </div>
        </div> 
    );
}